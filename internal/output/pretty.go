package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

// ANSI color codes
const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	BrightRed    = "\033[91m"
	BrightGreen  = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue   = "\033[94m"
	BrightCyan   = "\033[96m"
)

// PrettyFormatter outputs data with colors and nice formatting
type PrettyFormatter struct{}

// FormatHome formats a single home with colors
func (f *PrettyFormatter) FormatHome(home *models.HomeResponse) string {
	var sb strings.Builder

	title := home.AppNickname
	if title == "" {
		title = home.Address.Address1
	}
	if title == "" {
		title = "Home"
	}

	sb.WriteString(fmt.Sprintf("\n%s%s %s%s\n", Bold, Cyan, title, Reset))
	sb.WriteString(fmt.Sprintf("%s%s%s\n\n", Dim, strings.Repeat("─", len(title)+2), Reset))

	// Address
	if home.Address.Address1 != "" {
		sb.WriteString(fmt.Sprintf("  %s📍 Address%s\n", Bold, Reset))
		sb.WriteString(fmt.Sprintf("     %s\n", home.Address.Address1))
		if home.Address.PostalCode != "" || home.Address.City != "" {
			sb.WriteString(fmt.Sprintf("     %s %s, %s\n", home.Address.PostalCode, home.Address.City, home.Address.Country))
		}
		sb.WriteString("\n")
	}

	// Details
	sb.WriteString(fmt.Sprintf("  %s🏠 Details%s\n", Bold, Reset))
	if home.Size > 0 {
		sb.WriteString(fmt.Sprintf("     Size:      %s%d m²%s\n", BrightCyan, home.Size, Reset))
	}
	if home.Type != "" {
		sb.WriteString(fmt.Sprintf("     Type:      %s\n", home.Type))
	}
	if home.NumberOfResidents > 0 {
		sb.WriteString(fmt.Sprintf("     Residents: %d\n", home.NumberOfResidents))
	}
	if home.MainFuseSize > 0 {
		sb.WriteString(fmt.Sprintf("     Main Fuse: %d A\n", home.MainFuseSize))
	}
	sb.WriteString("\n")

	// Pulse status
	sb.WriteString(fmt.Sprintf("  %s⚡ Pulse%s\n", Bold, Reset))
	if home.Features.RealTimeConsumptionEnabled {
		sb.WriteString(fmt.Sprintf("     Status: %s● Connected%s\n", BrightGreen, Reset))
	} else {
		sb.WriteString(fmt.Sprintf("     Status: %s○ Not connected%s\n", Dim, Reset))
	}

	sb.WriteString(fmt.Sprintf("\n  %sID: %s%s\n", Dim, home.ID, Reset))

	return sb.String()
}

// FormatHomes formats multiple homes
func (f *PrettyFormatter) FormatHomes(homes []models.HomeResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%s%s⚡ Tibber Homes%s\n", Bold, Cyan, Reset))
	sb.WriteString(fmt.Sprintf("%s%s%s\n", Dim, strings.Repeat("─", 16), Reset))

	for _, home := range homes {
		sb.WriteString(f.FormatHome(&home))
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatPrices formats price info with colors
func (f *PrettyFormatter) FormatPrices(prices *models.PriceInfo, homeID string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%s%s⚡ Electricity Prices%s\n", Bold, Cyan, Reset))
	sb.WriteString(fmt.Sprintf("%s%s%s\n\n", Dim, strings.Repeat("─", 22), Reset))

	// Current price - big and prominent
	if prices.Current != nil {
		sb.WriteString(fmt.Sprintf("  %s%sNOW%s  ", Bold, BrightYellow, Reset))
		sb.WriteString(fmt.Sprintf("%s%s%.2f %s/kWh%s", Bold, priceColor(prices.Current.Level), prices.Current.Total, prices.Current.Currency, Reset))
		sb.WriteString(fmt.Sprintf("  %s\n\n", levelLabel(prices.Current.Level)))
	}

	// Today's prices
	if len(prices.Today) > 0 {
		sb.WriteString(fmt.Sprintf("  %s📅 Today%s\n", Bold, Reset))
		sb.WriteString(f.formatPriceList(prices.Today))
		sb.WriteString("\n")
	}

	// Tomorrow's prices
	if len(prices.Tomorrow) > 0 {
		sb.WriteString(fmt.Sprintf("  %s📅 Tomorrow%s\n", Bold, Reset))
		sb.WriteString(f.formatPriceList(prices.Tomorrow))
	} else {
		sb.WriteString(fmt.Sprintf("  %s📅 Tomorrow%s\n", Bold, Reset))
		sb.WriteString(fmt.Sprintf("     %sNot yet available (published ~13:00)%s\n", Dim, Reset))
	}

	return sb.String()
}

func (f *PrettyFormatter) formatPriceList(prices []models.Price) string {
	var sb strings.Builder

	// Find min/max for highlighting
	var minPrice, maxPrice float64 = prices[0].Total, prices[0].Total
	for _, p := range prices {
		if p.Total < minPrice {
			minPrice = p.Total
		}
		if p.Total > maxPrice {
			maxPrice = p.Total
		}
	}

	for _, p := range prices {
		hour := p.StartsAt.Local().Format("15:04")

		// Highlight current hour
		now := time.Now()
		isCurrent := p.StartsAt.Local().Hour() == now.Hour() &&
			p.StartsAt.Local().Day() == now.Day()

		prefix := "  "
		if isCurrent {
			prefix = fmt.Sprintf("%s▶%s ", BrightYellow, Reset)
		}

		// Price bar visualization
		barWidth := 20
		if maxPrice > minPrice {
			barLen := int(float64(barWidth) * (p.Total - minPrice) / (maxPrice - minPrice))
			if barLen < 1 {
				barLen = 1
			}
			bar := strings.Repeat("█", barLen) + strings.Repeat("░", barWidth-barLen)
			sb.WriteString(fmt.Sprintf("   %s%s %s%s%.2f%s %s%s%s\n",
				prefix, hour,
				priceColor(p.Level), bar, p.Total, Reset,
				Dim, p.Currency, Reset))
		} else {
			sb.WriteString(fmt.Sprintf("   %s%s %.2f %s\n", prefix, hour, p.Total, p.Currency))
		}
	}

	return sb.String()
}

// FormatLiveMeasurement formats live data with colors
func (f *PrettyFormatter) FormatLiveMeasurement(m *models.LiveMeasurement) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%s%s⚡ Live Power%s\n", Bold, Cyan, Reset))
	sb.WriteString(fmt.Sprintf("%s%s%s\n\n", Dim, strings.Repeat("─", 14), Reset))

	// Power - big and prominent
	powerColor := BrightGreen
	if m.Power > 5000 {
		powerColor = BrightRed
	} else if m.Power > 2000 {
		powerColor = BrightYellow
	}

	sb.WriteString(fmt.Sprintf("  %s%s%.0f W%s\n\n", Bold, powerColor, m.Power, Reset))

	// Production if any
	if m.PowerProduction > 0 {
		sb.WriteString(fmt.Sprintf("  %s☀️  Production:%s %.0f W\n", Green, Reset, m.PowerProduction))
	}

	// Today's stats
	sb.WriteString(fmt.Sprintf("  %s📊 Today%s\n", Bold, Reset))
	sb.WriteString(fmt.Sprintf("     Consumed: %s%.2f kWh%s\n", BrightCyan, m.AccumulatedConsumption, Reset))
	sb.WriteString(fmt.Sprintf("     Cost:     %s%.2f %s%s\n", BrightYellow, m.AccumulatedCost, m.Currency, Reset))

	// Voltage and current if available
	if m.VoltagePhase1 > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s🔌 Grid%s\n", Bold, Reset))
		sb.WriteString(fmt.Sprintf("     Voltage: %.0f / %.0f / %.0f V\n",
			m.VoltagePhase1, m.VoltagePhase2, m.VoltagePhase3))
		sb.WriteString(fmt.Sprintf("     Current: %.1f / %.1f / %.1f A\n",
			m.CurrentL1, m.CurrentL2, m.CurrentL3))
	}

	// Timestamp
	sb.WriteString(fmt.Sprintf("\n  %s%s%s\n", Dim, m.Timestamp.Local().Format("15:04:05"), Reset))

	return sb.String()
}

// Helper functions

func priceColor(level string) string {
	switch level {
	case "VERY_CHEAP":
		return BrightGreen
	case "CHEAP":
		return Green
	case "NORMAL":
		return Yellow
	case "EXPENSIVE":
		return Red
	case "VERY_EXPENSIVE":
		return BrightRed
	default:
		return Reset
	}
}

func levelLabel(level string) string {
	switch level {
	case "VERY_CHEAP":
		return fmt.Sprintf("%s● Very Cheap%s", BrightGreen, Reset)
	case "CHEAP":
		return fmt.Sprintf("%s● Cheap%s", Green, Reset)
	case "NORMAL":
		return fmt.Sprintf("%s● Normal%s", Yellow, Reset)
	case "EXPENSIVE":
		return fmt.Sprintf("%s● Expensive%s", Red, Reset)
	case "VERY_EXPENSIVE":
		return fmt.Sprintf("%s● Very Expensive%s", BrightRed, Reset)
	default:
		return level
	}
}
