package models

import "time"

// Home represents a Tibber home/residence
type Home struct {
	ID                   string   `json:"id"`
	AppNickname          string   `json:"appNickname"`
	Size                 int      `json:"size"`
	Type                 string   `json:"type"`
	NumberOfResidents    int      `json:"numberOfResidents"`
	PrimaryHeatingSource string   `json:"primaryHeatingSource"`
	HasVentilationSystem bool     `json:"hasVentilationSystem"`
	MainFuseSize         int      `json:"mainFuseSize"`
	Address              Address  `json:"address"`
	Features             Features `json:"features"`
}

// Address represents a physical address
type Address struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	Address3   string `json:"address3"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Country    string `json:"country"`
	Latitude   string `json:"latitude"`
	Longitude  string `json:"longitude"`
}

// Features represents home features
type Features struct {
	RealTimeConsumptionEnabled bool `json:"realTimeConsumptionEnabled"`
}

// Price represents an electricity price point
type Price struct {
	Total    float64   `json:"total"`
	Energy   float64   `json:"energy"`
	Tax      float64   `json:"tax"`
	StartsAt time.Time `json:"startsAt"`
	Level    string    `json:"level"`
	Currency string    `json:"currency"`
}

// PriceInfo contains current and upcoming prices
type PriceInfo struct {
	Current  *Price  `json:"current"`
	Today    []Price `json:"today"`
	Tomorrow []Price `json:"tomorrow"`
}

// LiveMeasurement represents real-time power data from Pulse
type LiveMeasurement struct {
	Timestamp              time.Time `json:"timestamp"`
	Power                  float64   `json:"power"`
	PowerProduction        float64   `json:"powerProduction"`
	AccumulatedConsumption float64   `json:"accumulatedConsumption"`
	AccumulatedProduction  float64   `json:"accumulatedProduction"`
	AccumulatedCost        float64   `json:"accumulatedCost"`
	AccumulatedReward      float64   `json:"accumulatedReward"`
	MinPower               float64   `json:"minPower"`
	MaxPower               float64   `json:"maxPower"`
	AveragePower           float64   `json:"averagePower"`
	VoltagePhase1          float64   `json:"voltagePhase1"`
	VoltagePhase2          float64   `json:"voltagePhase2"`
	VoltagePhase3          float64   `json:"voltagePhase3"`
	CurrentL1              float64   `json:"currentL1"`
	CurrentL2              float64   `json:"currentL2"`
	CurrentL3              float64   `json:"currentL3"`
	Currency               string    `json:"currency"`
}

// Viewer is the root GraphQL response type
type Viewer struct {
	Homes []HomeResponse `json:"homes"`
}

// HomeResponse is the GraphQL home response with subscription
type HomeResponse struct {
	Home
	CurrentSubscription *Subscription `json:"currentSubscription"`
}

// Subscription contains price info
type Subscription struct {
	PriceInfo *PriceInfo `json:"priceInfo"`
}
