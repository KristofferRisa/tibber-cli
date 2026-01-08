package output

import (
	"encoding/json"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

// JSONFormatter outputs data as JSON
type JSONFormatter struct{}

// FormatHome formats a single home as JSON
func (f *JSONFormatter) FormatHome(home *models.HomeResponse) string {
	data, _ := json.MarshalIndent(home, "", "  ")
	return string(data)
}

// FormatHomes formats multiple homes as JSON
func (f *JSONFormatter) FormatHomes(homes []models.HomeResponse) string {
	data, _ := json.MarshalIndent(homes, "", "  ")
	return string(data)
}

// FormatPrices formats price info as JSON
func (f *JSONFormatter) FormatPrices(prices *models.PriceInfo, homeID string) string {
	data, _ := json.MarshalIndent(prices, "", "  ")
	return string(data)
}

// FormatLiveMeasurement formats live data as compact JSON (for streaming)
func (f *JSONFormatter) FormatLiveMeasurement(m *models.LiveMeasurement) string {
	data, _ := json.Marshal(m)
	return string(data)
}
