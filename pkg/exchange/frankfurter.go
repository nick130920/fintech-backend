package exchange

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const frankfurterBaseURL = "https://api.frankfurter.dev/v2"

type FrankfurterProvider struct {
	client  *http.Client
	baseURL string
}

func NewFrankfurterProvider() *FrankfurterProvider {
	return &FrankfurterProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: frankfurterBaseURL,
	}
}

type frankfurterRateResponse struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

type frankfurterRatesResponse []frankfurterRateResponse

func (f *FrankfurterProvider) GetRate(base, quote string) (*Rate, error) {
	url := fmt.Sprintf("%s/rate/%s/%s", f.baseURL, strings.ToUpper(base), strings.ToUpper(quote))

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("frankfurter request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frankfurter returned status %d", resp.StatusCode)
	}

	var data frankfurterRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("frankfurter decode failed: %w", err)
	}

	return &Rate{
		Date:  data.Date,
		Base:  data.Base,
		Quote: data.Quote,
		Rate:  data.Rate,
	}, nil
}

func (f *FrankfurterProvider) GetRates(base string, quotes []string) ([]Rate, error) {
	url := fmt.Sprintf("%s/rates?base=%s", f.baseURL, strings.ToUpper(base))
	if len(quotes) > 0 {
		url += "&quotes=" + strings.Join(quotes, ",")
	}

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("frankfurter request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frankfurter returned status %d", resp.StatusCode)
	}

	var data frankfurterRatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("frankfurter decode failed: %w", err)
	}

	rates := make([]Rate, 0, len(data))
	for _, r := range data {
		rates = append(rates, Rate{
			Date:  r.Date,
			Base:  r.Base,
			Quote: r.Quote,
			Rate:  r.Rate,
		})
	}

	return rates, nil
}

func (f *FrankfurterProvider) GetHistoricalRate(base, quote string, date time.Time) (*Rate, error) {
	dateStr := date.Format("2006-01-02")
	url := fmt.Sprintf("%s/rate/%s/%s/%s", f.baseURL, strings.ToUpper(base), strings.ToUpper(quote), dateStr)

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("frankfurter request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frankfurter returned status %d", resp.StatusCode)
	}

	var data frankfurterRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("frankfurter decode failed: %w", err)
	}

	return &Rate{
		Date:  data.Date,
		Base:  data.Base,
		Quote: data.Quote,
		Rate:  data.Rate,
	}, nil
}
