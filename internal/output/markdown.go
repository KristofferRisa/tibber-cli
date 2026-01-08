package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

// MarkdownFormatter outputs data as Markdown tables
type MarkdownFormatter struct{}

// FormatHome formats a single home as Markdown
func (f *MarkdownFormatter) FormatHome(home *models.HomeResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s\n\n", homeTitle(home)))

	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| ID | `%s` |\n", home.ID))

	if home.Address.Address1 != "" {
		sb.WriteString(fmt.Sprintf("| Address | %s |\n", formatAddress(&home.Address)))
	}
	if home.Size > 0 {
		sb.WriteString(fmt.Sprintf("| Size | %d m² |\n", home.Size))
	}
	if home.Type != "" {
		sb.WriteString(fmt.Sprintf("| Type | %s |\n", home.Type))
	}
	if home.NumberOfResidents > 0 {
		sb.WriteString(fmt.Sprintf("| Residents | %d |\n", home.NumberOfResidents))
	}
	if home.MainFuseSize > 0 {
		sb.WriteString(fmt.Sprintf("| Main Fuse | %d A |\n", home.MainFuseSize))
	}

	pulseStatus := "No"
	if home.Features.RealTimeConsumptionEnabled {
		pulseStatus = "Yes"
	}
	sb.WriteString(fmt.Sprintf("| Pulse Enabled | %s |\n", pulseStatus))

	return sb.String()
}

// FormatHomes formats multiple homes as Markdown
func (f *MarkdownFormatter) FormatHomes(homes []models.HomeResponse) string {
	var sb strings.Builder

	sb.WriteString("# Tibber Homes\n\n")

	for i, home := range homes {
		sb.WriteString(f.FormatHome(&home))
		if i < len(homes)-1 {
			sb.WriteString("\n---\n\n")
		}
	}

	return sb.String()
}

// FormatPrices formats price info as Markdown
func (f *MarkdownFormatter) FormatPrices(prices *models.PriceInfo, homeID string) string {
	var sb strings.Builder

	sb.WriteString("# Electricity Prices\n\n")

	// Current price
	if prices.Current != nil {
		sb.WriteString("## Current Price\n\n")
		sb.WriteString(fmt.Sprintf("**%.2f %s/kWh** (%s)\n\n",
			prices.Current.Total,
			prices.Current.Currency,
			levelEmoji(prices.Current.Level)))
	}

	// Today's prices
	if len(prices.Today) > 0 {
		sb.WriteString("## Today\n\n")
		sb.WriteString(formatPriceTable(prices.Today))
		sb.WriteString("\n")
	}

	// Tomorrow's prices
	if len(prices.Tomorrow) > 0 {
		sb.WriteString("## Tomorrow\n\n")
		sb.WriteString(formatPriceTable(prices.Tomorrow))
	} else {
		sb.WriteString("*Tomorrow's prices not yet available (published around 13:00)*\n")
	}

	return sb.String()
}

// FormatLiveMeasurement formats live data as Markdown
func (f *MarkdownFormatter) FormatLiveMeasurement(m *models.LiveMeasurement) string {
	var sb strings.Builder

	sb.WriteString("## Live Power\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Power | %.0f W |\n", m.Power))
	if m.PowerProduction > 0 {
		sb.WriteString(fmt.Sprintf("| Production | %.0f W |\n", m.PowerProduction))
	}
	sb.WriteString(fmt.Sprintf("| Today | %.2f kWh |\n", m.AccumulatedConsumption))
	sb.WriteString(fmt.Sprintf("| Cost | %.2f %s |\n", m.AccumulatedCost, m.Currency))

	if m.VoltagePhase1 > 0 {
		sb.WriteString(fmt.Sprintf("| Voltage | %.1f / %.1f / %.1f V |\n",
			m.VoltagePhase1, m.VoltagePhase2, m.VoltagePhase3))
	}
	if m.CurrentL1 > 0 {
		sb.WriteString(fmt.Sprintf("| Current | %.1f / %.1f / %.1f A |\n",
			m.CurrentL1, m.CurrentL2, m.CurrentL3))
	}

	sb.WriteString(fmt.Sprintf("| Updated | %s |\n", m.Timestamp.Format(time.RFC3339)))

	return sb.String()
}

// Helper functions

func homeTitle(home *models.HomeResponse) string {
	if home.AppNickname != "" {
		return home.AppNickname
	}
	if home.Address.Address1 != "" {
		return home.Address.Address1
	}
	return "Home"
}

func formatAddress(addr *models.Address) string {
	parts := []string{}
	if addr.Address1 != "" {
		parts = append(parts, addr.Address1)
	}
	if addr.PostalCode != "" || addr.City != "" {
		parts = append(parts, fmt.Sprintf("%s %s", addr.PostalCode, addr.City))
	}
	return strings.Join(parts, ", ")
}

func formatPriceTable(prices []models.Price) string {
	var sb strings.Builder

	sb.WriteString("| Time | Price | Level |\n")
	sb.WriteString("|------|-------|-------|\n")

	for _, p := range prices {
		hour := p.StartsAt.Local().Format("15:04")
		sb.WriteString(fmt.Sprintf("| %s | %.2f %s | %s |\n",
			hour, p.Total, p.Currency, levelEmoji(p.Level)))
	}

	return sb.String()
}

func levelEmoji(level string) string {
	switch level {
	case "VERY_CHEAP":
		return "VERY_CHEAP"
	case "CHEAP":
		return "CHEAP"
	case "NORMAL":
		return "NORMAL"
	case "EXPENSIVE":
		return "EXPENSIVE"
	case "VERY_EXPENSIVE":
		return "VERY_EXPENSIVE"
	default:
		return level
	}
}
