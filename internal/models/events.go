package models

import "time"

type EventType string

const (
	OrderPlaced     EventType = "ORDER_PLACED"
	OrderCompleted  EventType = "ORDER_COMPLETED"
	OrderFailed     EventType = "ORDER_FAILED"
	PaymentSuccess  EventType = "PAYMENT_SUCCESS"
	PaymentFailed   EventType = "PAYMENT_FAILED"
	UserSignup      EventType = "USER_SIGNUP"
	VoucherRedeemed EventType = "VOUCHER_REDEEMED"
)

type Region string

const (
	RegionID Region = "ID"
	RegionSG Region = "SG"
	RegionMY Region = "MY"
	RegionTH Region = "TH"
	RegionVN Region = "VN"
	RegionPH Region = "PH"
	RegionTW Region = "TW"
	RegionBR Region = "BR"
)

var AllRegions = []Region{
	RegionID, RegionSG, RegionMY, RegionTH,
	RegionVN, RegionPH, RegionTW, RegionBR,
}

type Event struct {
	ID        string            `json:"id"`
	Type      EventType         `json:"type"`
	Region    Region            `json:"region"`
	UserID    string            `json:"user_id"`
	Amount    float64           `json:"amount"`
	Currency  string            `json:"currency"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

type RegionStats struct {
	Region       Region  `json:"region"`
	OrderCount   int64   `json:"order_count"`
	FailureCount int64   `json:"failure_count"`
	Revenue      float64 `json:"revenue"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}
