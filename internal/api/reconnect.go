package api

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/coder/websocket"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

// Reconnect defaults. MaxPerHour is the one that matters: Tibber allows 20
// WebSocket connections per hour per token, so a stream that keeps failing has
// to leave headroom rather than spend the whole allowance and lock the user out
// for the rest of the hour.
const (
	defaultBaseDelay    = 5 * time.Second
	defaultMaxDelay     = 5 * time.Minute
	defaultMaxPerHour   = 10
	defaultHealthyAfter = time.Minute
)

// statusInvalidToken is the WebSocket close code Tibber sends for a rejected token.
const statusInvalidToken websocket.StatusCode = 4403

// ErrReconnectBudget reports that the per-hour connection cap was reached.
// Retrying past it would burn the Tibber quota that a manual restart needs.
var ErrReconnectBudget = errors.New("reconnect budget exhausted")

// ReconnectPolicy bounds how a dropped live stream is retried.
type ReconnectPolicy struct {
	// BaseDelay is the wait before the first retry. It doubles on each
	// consecutive failure, up to MaxDelay.
	BaseDelay time.Duration

	// MaxDelay caps the backoff.
	MaxDelay time.Duration

	// MaxPerHour caps connection attempts in any rolling hour, counting the
	// first one. Reaching it ends the stream with ErrReconnectBudget.
	MaxPerHour int

	// HealthyAfter is how long a connection must stay up, having delivered at
	// least one measurement, before the backoff resets to BaseDelay.
	HealthyAfter time.Duration

	// Notify, if set, runs before each backoff wait. It must not write to
	// stdout: that stream carries measurement data.
	Notify func(attempt int, delay time.Duration, cause error)

	// Test seams. Left nil, they resolve to the real clock and jittered delays.
	now    func() time.Time
	sleep  func(context.Context, time.Duration) error
	jitter func(time.Duration) time.Duration
}

// DefaultReconnectPolicy returns the policy used by `powerctl live`.
func DefaultReconnectPolicy() ReconnectPolicy {
	return ReconnectPolicy{
		BaseDelay:    defaultBaseDelay,
		MaxDelay:     defaultMaxDelay,
		MaxPerHour:   defaultMaxPerHour,
		HealthyAfter: defaultHealthyAfter,
	}
}

func (p ReconnectPolicy) withDefaults() ReconnectPolicy {
	if p.BaseDelay <= 0 {
		p.BaseDelay = defaultBaseDelay
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = defaultMaxDelay
	}
	if p.MaxDelay < p.BaseDelay {
		p.MaxDelay = p.BaseDelay
	}
	if p.MaxPerHour <= 0 {
		p.MaxPerHour = defaultMaxPerHour
	}
	if p.HealthyAfter <= 0 {
		p.HealthyAfter = defaultHealthyAfter
	}
	if p.now == nil {
		p.now = time.Now
	}
	if p.sleep == nil {
		p.sleep = sleepCtx
	}
	if p.jitter == nil {
		p.jitter = equalJitter
	}
	return p
}

// SubscribeWithReconnect streams live measurements, reconnecting on transient
// failures within the bounds of p. It returns nil when the server ends the
// subscription, ctx's error when the caller cancels, and otherwise the failure
// that stopped it: a fatal error, or ErrReconnectBudget.
func (c *LiveClient) SubscribeWithReconnect(ctx context.Context, p ReconnectPolicy, handler func(*models.LiveMeasurement) error) error {
	return retryLoop(ctx, p, func(ctx context.Context) (bool, error) {
		delivered := false
		err := c.Subscribe(ctx, func(m *models.LiveMeasurement) error {
			delivered = true
			return handler(m)
		})
		return delivered, err
	})
}

// attemptFunc runs one connection attempt, reporting whether it delivered at
// least one measurement before failing.
type attemptFunc func(ctx context.Context) (delivered bool, err error)

func retryLoop(ctx context.Context, p ReconnectPolicy, attempt attemptFunc) error {
	p = p.withDefaults()

	var window []time.Time // connection attempts inside the last hour
	consecutive := 0

	for {
		started := p.now()
		window = append(pruneWindow(window, started), started)

		delivered, err := attempt(ctx)
		if err == nil {
			return nil
		}
		// Cancellation is the caller's decision, so it outranks whatever error
		// the attempt returned while unwinding.
		if ctx.Err() != nil {
			return err
		}
		if !retryable(err) {
			return err
		}

		// A connection that carried data for a while was not part of this run of
		// failures, so the next retry starts over at BaseDelay.
		if delivered && p.now().Sub(started) >= p.HealthyAfter {
			consecutive = 0
		}
		consecutive++

		delay := p.jitter(backoff(p.BaseDelay, p.MaxDelay, consecutive-1))

		// Check the budget as it will stand once the wait ends: waiting can age an
		// attempt out of the window and free a slot, but when it cannot there is
		// no reason to pause before saying so.
		if spent := len(pruneWindow(window, p.now().Add(delay))); spent >= p.MaxPerHour {
			return fmt.Errorf("%w: %d connection attempts in the last hour (Tibber allows 20); last error: %v",
				ErrReconnectBudget, spent, err)
		}

		if p.Notify != nil {
			p.Notify(consecutive, delay, err)
		}
		if err := p.sleep(ctx, delay); err != nil {
			return err
		}
	}
}

// retryable reports whether reconnecting could plausibly fix err. Cancellation
// is not considered here; retryLoop checks the context directly, so an internal
// deadline (the ack timeout) stays retryable.
//
// Being too permissive here is expensive: an invalid token or an unknown home ID
// fails identically on every attempt, so retrying only spends the hourly budget
// before handing the user the same message.
func retryable(err error) bool {
	if err == nil {
		return false
	}
	if IsFatal(err) {
		return false
	}
	if websocket.CloseStatus(err) == statusInvalidToken {
		return false
	}
	return true
}

// fatalError marks a stream failure that reconnecting cannot fix.
type fatalError struct{ err error }

func (e *fatalError) Error() string { return e.err.Error() }
func (e *fatalError) Unwrap() error { return e.err }

func markFatal(err error) error {
	if err == nil {
		return nil
	}
	return &fatalError{err: err}
}

// IsFatal reports whether err ends the live stream for good.
func IsFatal(err error) bool {
	var fe *fatalError
	return errors.As(err, &fe)
}

// backoff returns base doubled n times, capped at ceiling.
func backoff(base, ceiling time.Duration, n int) time.Duration {
	d := base
	for i := 0; i < n; i++ {
		d *= 2
		if d >= ceiling || d <= 0 { // d <= 0 means the doubling overflowed
			return ceiling
		}
	}
	return d
}

// equalJitter keeps half the delay and randomises the rest, so several clients
// dropped by the same outage spread their retries instead of arriving together.
func equalJitter(d time.Duration) time.Duration {
	half := d / 2
	if half <= 0 {
		return d
	}
	return half + time.Duration(rand.Int64N(int64(half)))
}

// sleepCtx waits for d, returning early if ctx is cancelled. Ctrl+C during a
// backoff wait has to exit now, not five minutes from now.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// pruneWindow drops attempts older than an hour. Entries are appended in order,
// so the stale ones are always a prefix.
func pruneWindow(window []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-time.Hour)
	i := 0
	for i < len(window) && !window[i].After(cutoff) {
		i++
	}
	return window[i:]
}
