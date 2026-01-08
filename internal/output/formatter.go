package output

import "github.com/kristofferrisa/powerctl-cli/internal/models"

// Formatter defines the interface for output formatting
type Formatter interface {
	FormatHome(home *models.HomeResponse) string
	FormatHomes(homes []models.HomeResponse) string
	FormatPrices(prices *models.PriceInfo, homeID string) string
	FormatLiveMeasurement(m *models.LiveMeasurement) string
}

// New creates a formatter based on the format name
func New(format string) Formatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "markdown", "md":
		return &MarkdownFormatter{}
	case "pretty", "":
		return &PrettyFormatter{}
	default:
		return &PrettyFormatter{}
	}
}
