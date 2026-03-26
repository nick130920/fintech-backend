package exchange

import "time"

type Rate struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

type Provider interface {
	GetRate(base, quote string) (*Rate, error)
	GetRates(base string, quotes []string) ([]Rate, error)
	GetHistoricalRate(base, quote string, date time.Time) (*Rate, error)
}
