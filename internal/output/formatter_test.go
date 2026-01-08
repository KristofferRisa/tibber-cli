package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

func TestNew_ReturnsCorrectFormatter(t *testing.T) {
	tests := []struct {
		format   string
		wantType string
	}{
		{"json", "JSONFormatter"},
		{"markdown", "MarkdownFormatter"},
		{"md", "MarkdownFormatter"},
		{"pretty", "PrettyFormatter"},
		{"", "PrettyFormatter"},
		{"unknown", "PrettyFormatter"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f := New(tt.format)

			switch tt.wantType {
			case "JSONFormatter":
				if _, ok := f.(*JSONFormatter); !ok {
					t.Errorf("New(%q) = %T, want *JSONFormatter", tt.format, f)
				}
			case "MarkdownFormatter":
				if _, ok := f.(*MarkdownFormatter); !ok {
					t.Errorf("New(%q) = %T, want *MarkdownFormatter", tt.format, f)
				}
			case "PrettyFormatter":
				if _, ok := f.(*PrettyFormatter); !ok {
					t.Errorf("New(%q) = %T, want *PrettyFormatter", tt.format, f)
				}
			}
		})
	}
}

func sampleHome() *models.HomeResponse {
	return &models.HomeResponse{
		Home: models.Home{
			ID:          "home-123",
			AppNickname: "My House",
			Size:        150,
			Type:        "HOUSE",
			Address: models.Address{
				Address1:   "123 Main St",
				PostalCode: "12345",
				City:       "Oslo",
				Country:    "Norway",
			},
			Features: models.Features{
				RealTimeConsumptionEnabled: true,
			},
		},
	}
}

func samplePrices() *models.PriceInfo {
	now := time.Now()
	return &models.PriceInfo{
		Current: &models.Price{
			Total:    0.45,
			Energy:   0.35,
			Tax:      0.10,
			StartsAt: now,
			Level:    "NORMAL",
			Currency: "NOK",
		},
		Today: []models.Price{
			{Total: 0.40, Level: "CHEAP", StartsAt: now.Add(-1 * time.Hour), Currency: "NOK"},
			{Total: 0.45, Level: "NORMAL", StartsAt: now, Currency: "NOK"},
			{Total: 0.60, Level: "EXPENSIVE", StartsAt: now.Add(1 * time.Hour), Currency: "NOK"},
		},
	}
}

func sampleLiveMeasurement() *models.LiveMeasurement {
	return &models.LiveMeasurement{
		Timestamp:              time.Now(),
		Power:                  1234,
		AccumulatedConsumption: 12.5,
		AccumulatedCost:        45.30,
		VoltagePhase1:          230,
		VoltagePhase2:          231,
		VoltagePhase3:          229,
		CurrentL1:              5.2,
		CurrentL2:              3.1,
		CurrentL3:              4.5,
		Currency:               "NOK",
	}
}

// JSON Formatter Tests

func TestJSONFormatter_FormatHome(t *testing.T) {
	f := &JSONFormatter{}
	home := sampleHome()

	output := f.FormatHome(home)

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("FormatHome() output is not valid JSON: %v", err)
	}

	// Check key fields
	if result["id"] != "home-123" {
		t.Errorf("FormatHome() id = %v, want home-123", result["id"])
	}
	if result["appNickname"] != "My House" {
		t.Errorf("FormatHome() appNickname = %v, want My House", result["appNickname"])
	}
}

func TestJSONFormatter_FormatPrices(t *testing.T) {
	f := &JSONFormatter{}
	prices := samplePrices()

	output := f.FormatPrices(prices, "home-123")

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("FormatPrices() output is not valid JSON: %v", err)
	}

	// Check current price exists
	if result["current"] == nil {
		t.Error("FormatPrices() missing current price")
	}
}

func TestJSONFormatter_FormatLiveMeasurement(t *testing.T) {
	f := &JSONFormatter{}
	m := sampleLiveMeasurement()

	output := f.FormatLiveMeasurement(m)

	// Should be valid JSON (compact, single line)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("FormatLiveMeasurement() output is not valid JSON: %v", err)
	}

	// Should not contain newlines (compact format for streaming)
	if strings.Contains(output, "\n") {
		t.Error("FormatLiveMeasurement() should be compact (no newlines)")
	}

	// Check power value
	if result["power"].(float64) != 1234 {
		t.Errorf("FormatLiveMeasurement() power = %v, want 1234", result["power"])
	}
}

// Markdown Formatter Tests

func TestMarkdownFormatter_FormatHome(t *testing.T) {
	f := &MarkdownFormatter{}
	home := sampleHome()

	output := f.FormatHome(home)

	// Should contain key elements
	if !strings.Contains(output, "My House") {
		t.Error("FormatHome() should contain home nickname")
	}
	if !strings.Contains(output, "home-123") {
		t.Error("FormatHome() should contain home ID")
	}
	if !strings.Contains(output, "150") {
		t.Error("FormatHome() should contain size")
	}
	if !strings.Contains(output, "|") {
		t.Error("FormatHome() should contain table formatting")
	}
}

func TestMarkdownFormatter_FormatPrices(t *testing.T) {
	f := &MarkdownFormatter{}
	prices := samplePrices()

	output := f.FormatPrices(prices, "home-123")

	// Should contain headers
	if !strings.Contains(output, "# ") {
		t.Error("FormatPrices() should contain markdown headers")
	}
	// Should contain price value
	if !strings.Contains(output, "0.45") {
		t.Error("FormatPrices() should contain current price")
	}
	// Should contain table
	if !strings.Contains(output, "|") {
		t.Error("FormatPrices() should contain table formatting")
	}
}

func TestMarkdownFormatter_FormatLiveMeasurement(t *testing.T) {
	f := &MarkdownFormatter{}
	m := sampleLiveMeasurement()

	output := f.FormatLiveMeasurement(m)

	// Should contain power value
	if !strings.Contains(output, "1234") {
		t.Error("FormatLiveMeasurement() should contain power value")
	}
	// Should contain table
	if !strings.Contains(output, "|") {
		t.Error("FormatLiveMeasurement() should contain table formatting")
	}
}

// Pretty Formatter Tests

func TestPrettyFormatter_FormatHome(t *testing.T) {
	f := &PrettyFormatter{}
	home := sampleHome()

	output := f.FormatHome(home)

	// Should contain home name
	if !strings.Contains(output, "My House") {
		t.Error("FormatHome() should contain home nickname")
	}
	// Should contain ANSI color codes (escape sequences)
	if !strings.Contains(output, "\033[") {
		t.Error("FormatHome() should contain ANSI color codes")
	}
	// Should contain Pulse status
	if !strings.Contains(output, "Connected") {
		t.Error("FormatHome() should show Pulse as connected")
	}
}

func TestPrettyFormatter_FormatPrices(t *testing.T) {
	f := &PrettyFormatter{}
	prices := samplePrices()

	output := f.FormatPrices(prices, "home-123")

	// Should contain NOW indicator
	if !strings.Contains(output, "NOW") {
		t.Error("FormatPrices() should contain NOW indicator")
	}
	// Should contain color codes
	if !strings.Contains(output, "\033[") {
		t.Error("FormatPrices() should contain ANSI color codes")
	}
	// Should contain price bars
	if !strings.Contains(output, "█") || !strings.Contains(output, "░") {
		t.Error("FormatPrices() should contain price bar visualization")
	}
}

func TestPrettyFormatter_FormatLiveMeasurement(t *testing.T) {
	f := &PrettyFormatter{}
	m := sampleLiveMeasurement()

	output := f.FormatLiveMeasurement(m)

	// Should contain power with unit
	if !strings.Contains(output, "1234") || !strings.Contains(output, "W") {
		t.Error("FormatLiveMeasurement() should contain power with unit")
	}
	// Should contain color codes
	if !strings.Contains(output, "\033[") {
		t.Error("FormatLiveMeasurement() should contain ANSI color codes")
	}
}

func TestPrettyFormatter_PowerColorByUsage(t *testing.T) {
	f := &PrettyFormatter{}

	tests := []struct {
		name  string
		power float64
		color string
	}{
		{"low power", 500, BrightGreen},
		{"medium power", 3000, BrightYellow},
		{"high power", 6000, BrightRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &models.LiveMeasurement{
				Timestamp: time.Now(),
				Power:     tt.power,
				Currency:  "NOK",
			}
			output := f.FormatLiveMeasurement(m)

			if !strings.Contains(output, tt.color) {
				t.Errorf("Power %.0f should use color %q", tt.power, tt.color)
			}
		})
	}
}
