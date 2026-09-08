package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// fakeClock drives retryLoop without real waiting: sleeping advances the clock,
// so the rolling-hour window can be exercised in a few microseconds.
type fakeClock struct {
	t      time.Time
	delays []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time { return c.t }

func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func (c *fakeClock) sleep(_ context.Context, d time.Duration) error {
	c.delays = append(c.delays, d)
	c.advance(d)
	return nil
}

// testPolicy is the default policy wired to a fake clock, with jitter removed so
// delays are exact.
func testPolicy(c *fakeClock) ReconnectPolicy {
	p := DefaultReconnectPolicy()
	p.now = c.now
	p.sleep = c.sleep
	p.jitter = func(d time.Duration) time.Duration { return d }
	return p
}

// alwaysFails is an attempt that never succeeds. It aborts the test rather than
// spinning forever if the hourly budget stops capping attempts, so a regression
// there fails fast instead of hanging until the go test timeout.
func alwaysFails(t *testing.T, limit int, count *int) attemptFunc {
	t.Helper()
	return func(context.Context) (bool, error) {
		*count++
		if *count > limit {
			t.Fatalf("made %d attempts with no cap; the hourly budget is not holding", *count)
		}
		return false, errors.New("read error: connection reset")
	}
}

func TestRetryLoop_RetriesTransientFailures(t *testing.T) {
	clock := newFakeClock()
	attempts := 0

	err := retryLoop(context.Background(), testPolicy(clock), func(context.Context) (bool, error) {
		attempts++
		if attempts < 3 {
			return false, errors.New("read error: connection reset")
		}
		return true, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryLoop_StopsOnFatalError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "invalid home ID",
			err:  markFatal(errors.New("invalid or non-existing home ID")),
		},
		{
			name: "graphql error payload",
			err:  markFatal(errors.New("subscription error: something is wrong")),
		},
		{
			name: "invalid token close code",
			err:  fmt.Errorf("read error: %w", websocket.CloseError{Code: statusInvalidToken, Reason: "Invalid token"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := newFakeClock()
			attempts := 0

			err := retryLoop(context.Background(), testPolicy(clock), func(context.Context) (bool, error) {
				attempts++
				return false, tt.err
			})

			if !errors.Is(err, tt.err) {
				t.Fatalf("expected %v, got %v", tt.err, err)
			}
			if attempts != 1 {
				t.Errorf("expected the stream to stop after 1 attempt, got %d", attempts)
			}
			if len(clock.delays) != 0 {
				t.Errorf("expected no backoff waits, got %v", clock.delays)
			}
		})
	}
}

func TestRetryLoop_StaysUnderHourlyBudget(t *testing.T) {
	clock := newFakeClock()
	policy := testPolicy(clock)
	attempts := 0

	err := retryLoop(context.Background(), policy, alwaysFails(t, policy.MaxPerHour, &attempts))

	if !errors.Is(err, ErrReconnectBudget) {
		t.Fatalf("expected ErrReconnectBudget, got %v", err)
	}
	if attempts != policy.MaxPerHour {
		t.Errorf("expected %d attempts, got %d", policy.MaxPerHour, attempts)
	}
	// The point of the cap: Tibber allows 20 connections per hour and a manual
	// restart needs some of them left.
	if attempts >= 20 {
		t.Errorf("attempts %d would exhaust Tibber's hourly connection quota", attempts)
	}
	if elapsed := clock.now().Sub(newFakeClock().now()); elapsed > time.Hour {
		t.Errorf("budget should be spent inside one hour, took %v", elapsed)
	}
}

func TestRetryLoop_BudgetRecoversAfterAnHour(t *testing.T) {
	clock := newFakeClock()
	policy := testPolicy(clock)
	attempts := 0

	err := retryLoop(context.Background(), policy, func(context.Context) (bool, error) {
		attempts++
		// Every connection dies after two hours, so no two attempts ever share a
		// rolling-hour window and the budget never runs out.
		clock.advance(2 * time.Hour)
		if attempts == 4 {
			return true, nil
		}
		return true, errors.New("read error: connection reset")
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 4 {
		t.Errorf("expected 4 attempts, got %d", attempts)
	}
}

func TestRetryLoop_BackoffDoublesAndCaps(t *testing.T) {
	clock := newFakeClock()
	policy := testPolicy(clock)
	policy.MaxPerHour = 8

	attempts := 0
	_ = retryLoop(context.Background(), policy, alwaysFails(t, policy.MaxPerHour, &attempts))

	want := []time.Duration{
		5 * time.Second,
		10 * time.Second,
		20 * time.Second,
		40 * time.Second,
		80 * time.Second,
		160 * time.Second,
		defaultMaxDelay,
	}
	if len(clock.delays) != len(want) {
		t.Fatalf("expected %d waits, got %d: %v", len(want), len(clock.delays), clock.delays)
	}
	for i, w := range want {
		if clock.delays[i] != w {
			t.Errorf("wait %d: expected %v, got %v", i, w, clock.delays[i])
		}
	}
}

func TestRetryLoop_HealthyConnectionResetsBackoff(t *testing.T) {
	clock := newFakeClock()
	policy := testPolicy(clock)
	attempts := 0

	_ = retryLoop(context.Background(), policy, func(context.Context) (bool, error) {
		attempts++
		switch attempts {
		case 3:
			// Ran fine for a day before dropping, so this is a fresh outage
			// rather than a continuation of the previous run of failures.
			clock.advance(24 * time.Hour)
			return true, errors.New("read error: connection reset")
		case 4:
			return true, nil
		default:
			return false, errors.New("read error: connection reset")
		}
	})

	want := []time.Duration{5 * time.Second, 10 * time.Second, 5 * time.Second}
	if len(clock.delays) != len(want) {
		t.Fatalf("expected waits %v, got %v", want, clock.delays)
	}
	for i, w := range want {
		if clock.delays[i] != w {
			t.Errorf("wait %d: expected %v, got %v", i, w, clock.delays[i])
		}
	}
}

func TestRetryLoop_HealthyConnectionWithoutDataDoesNotReset(t *testing.T) {
	clock := newFakeClock()
	policy := testPolicy(clock)
	attempts := 0

	_ = retryLoop(context.Background(), policy, func(context.Context) (bool, error) {
		attempts++
		// Connects and stays open, but never delivers a measurement. That is not
		// a healthy stream, so the backoff must keep growing.
		clock.advance(24 * time.Hour)
		if attempts == 3 {
			return false, nil
		}
		return false, errors.New("read error: connection reset")
	})

	want := []time.Duration{5 * time.Second, 10 * time.Second}
	if len(clock.delays) != len(want) {
		t.Fatalf("expected waits %v, got %v", want, clock.delays)
	}
	for i, w := range want {
		if clock.delays[i] != w {
			t.Errorf("wait %d: expected %v, got %v", i, w, clock.delays[i])
		}
	}
}

func TestRetryLoop_CancelledContextStopsImmediately(t *testing.T) {
	t.Run("during an attempt", func(t *testing.T) {
		clock := newFakeClock()
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		err := retryLoop(ctx, testPolicy(clock), func(context.Context) (bool, error) {
			attempts++
			cancel()
			return true, fmt.Errorf("read error: %w", context.Canceled)
		})

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
		if len(clock.delays) != 0 {
			t.Errorf("expected no backoff wait after cancellation, got %v", clock.delays)
		}
	})

	t.Run("during a backoff wait", func(t *testing.T) {
		clock := newFakeClock()
		policy := testPolicy(clock)
		// Ctrl+C has to land during the wait, not five minutes later.
		policy.sleep = func(context.Context, time.Duration) error { return context.Canceled }
		attempts := 0

		err := retryLoop(context.Background(), policy, func(context.Context) (bool, error) {
			attempts++
			return false, errors.New("read error: connection reset")
		})

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})
}

func TestRetryLoop_NotifiesBeforeEachWait(t *testing.T) {
	clock := newFakeClock()
	policy := testPolicy(clock)
	policy.MaxPerHour = 3

	type notice struct {
		attempt int
		delay   time.Duration
		cause   string
	}
	var notices []notice
	policy.Notify = func(attempt int, delay time.Duration, cause error) {
		notices = append(notices, notice{attempt, delay, cause.Error()})
	}

	attempts := 0
	_ = retryLoop(context.Background(), policy, alwaysFails(t, policy.MaxPerHour, &attempts))

	want := []notice{
		{1, 5 * time.Second, "read error: connection reset"},
		{2, 10 * time.Second, "read error: connection reset"},
	}
	if len(notices) != len(want) {
		t.Fatalf("expected %d notices, got %d: %v", len(want), len(notices), notices)
	}
	for i, w := range want {
		if notices[i] != w {
			t.Errorf("notice %d: expected %+v, got %+v", i, w, notices[i])
		}
	}
}

func TestSleepCtx_ReturnsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if err := sleepCtx(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("expected an immediate return, waited %v", elapsed)
	}
}

func TestRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"read error", errors.New("read error: connection reset by peer"), true},
		{"abnormal close", fmt.Errorf("read error: %w", websocket.CloseError{Code: websocket.StatusAbnormalClosure}), true},
		{"going away", fmt.Errorf("read error: %w", websocket.CloseError{Code: websocket.StatusGoingAway}), true},
		{"ack timeout", fmt.Errorf("failed to read ack: %w", context.DeadlineExceeded), true},
		{"invalid token", fmt.Errorf("read error: %w", websocket.CloseError{Code: statusInvalidToken, Reason: "Invalid token"}), false},
		{"invalid home ID", markFatal(errors.New("invalid or non-existing home ID")), false},
		{"wrapped fatal", fmt.Errorf("stream: %w", markFatal(errors.New("subscription error"))), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryable(tt.err); got != tt.want {
				t.Errorf("retryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		n    int
		want time.Duration
	}{
		{0, 5 * time.Second},
		{1, 10 * time.Second},
		{4, 80 * time.Second},
		{6, 5 * time.Minute},
		{100, 5 * time.Minute}, // must cap rather than overflow
	}

	for _, tt := range tests {
		if got := backoff(defaultBaseDelay, defaultMaxDelay, tt.n); got != tt.want {
			t.Errorf("backoff(n=%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

func TestEqualJitter_StaysWithinHalfAndFull(t *testing.T) {
	const d = 40 * time.Second

	for i := 0; i < 200; i++ {
		got := equalJitter(d)
		if got < d/2 || got >= d {
			t.Fatalf("equalJitter(%v) = %v, want [%v, %v)", d, got, d/2, d)
		}
	}

	if got := equalJitter(0); got != 0 {
		t.Errorf("equalJitter(0) = %v, want 0", got)
	}
	if got := equalJitter(time.Nanosecond); got != time.Nanosecond {
		t.Errorf("equalJitter(1ns) = %v, want 1ns", got)
	}
}

func TestPruneWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	window := []time.Time{
		now.Add(-90 * time.Minute), // stale
		now.Add(-time.Hour),        // exactly an hour old: stale
		now.Add(-30 * time.Minute),
		now,
	}

	got := pruneWindow(window, now)
	if len(got) != 2 {
		t.Fatalf("expected 2 attempts left in the window, got %d", len(got))
	}
	if !got[0].Equal(now.Add(-30*time.Minute)) || !got[1].Equal(now) {
		t.Errorf("wrong entries kept: %v", got)
	}
}

func TestReconnectPolicy_WithDefaults(t *testing.T) {
	p := ReconnectPolicy{}.withDefaults()

	if p.BaseDelay != defaultBaseDelay || p.MaxDelay != defaultMaxDelay {
		t.Errorf("expected default delays, got base=%v max=%v", p.BaseDelay, p.MaxDelay)
	}
	if p.MaxPerHour != defaultMaxPerHour || p.HealthyAfter != defaultHealthyAfter {
		t.Errorf("expected default budget, got %d per hour, healthy after %v", p.MaxPerHour, p.HealthyAfter)
	}
	if p.now == nil || p.sleep == nil || p.jitter == nil {
		t.Error("expected the clock and jitter seams to be filled in")
	}

	// A ceiling below the base would make the backoff shrink.
	clamped := ReconnectPolicy{BaseDelay: time.Minute, MaxDelay: time.Second}.withDefaults()
	if clamped.MaxDelay != time.Minute {
		t.Errorf("expected MaxDelay clamped up to BaseDelay, got %v", clamped.MaxDelay)
	}
}
