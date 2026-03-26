package exchange

import (
	"sync"
	"time"
)

type cacheEntry struct {
	rates     []Rate
	fetchedAt time.Time
}

type CachedProvider struct {
	provider Provider
	ttl      time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

func NewCachedProvider(provider Provider, ttl time.Duration) *CachedProvider {
	return &CachedProvider{
		provider: provider,
		ttl:      ttl,
		cache:    make(map[string]cacheEntry),
	}
}

func (c *CachedProvider) GetRate(base, quote string) (*Rate, error) {
	key := "rate:" + base + ":" + quote

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < c.ttl && len(entry.rates) > 0 {
		return &entry.rates[0], nil
	}

	rate, err := c.provider.GetRate(base, quote)
	if err != nil {
		if ok && len(entry.rates) > 0 {
			return &entry.rates[0], nil
		}
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = cacheEntry{rates: []Rate{*rate}, fetchedAt: time.Now()}
	c.mu.Unlock()

	return rate, nil
}

func (c *CachedProvider) GetRates(base string, quotes []string) ([]Rate, error) {
	key := "rates:" + base

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < c.ttl && len(entry.rates) > 0 {
		if len(quotes) == 0 {
			return entry.rates, nil
		}
		quotesSet := make(map[string]bool, len(quotes))
		for _, q := range quotes {
			quotesSet[q] = true
		}
		filtered := make([]Rate, 0, len(quotes))
		for _, r := range entry.rates {
			if quotesSet[r.Quote] {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == len(quotes) {
			return filtered, nil
		}
	}

	rates, err := c.provider.GetRates(base, quotes)
	if err != nil {
		if ok && len(entry.rates) > 0 {
			return entry.rates, nil
		}
		return nil, err
	}

	c.mu.Lock()
	if len(quotes) == 0 {
		c.cache[key] = cacheEntry{rates: rates, fetchedAt: time.Now()}
	} else {
		existing := c.cache[key]
		merged := mergeRates(existing.rates, rates)
		c.cache[key] = cacheEntry{rates: merged, fetchedAt: time.Now()}
	}
	c.mu.Unlock()

	return rates, nil
}

func (c *CachedProvider) GetHistoricalRate(base, quote string, date time.Time) (*Rate, error) {
	key := "hist:" + base + ":" + quote + ":" + date.Format("2006-01-02")

	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && len(entry.rates) > 0 {
		return &entry.rates[0], nil
	}

	rate, err := c.provider.GetHistoricalRate(base, quote, date)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = cacheEntry{rates: []Rate{*rate}, fetchedAt: time.Now()}
	c.mu.Unlock()

	return rate, nil
}

func mergeRates(existing, incoming []Rate) []Rate {
	byQuote := make(map[string]Rate, len(existing)+len(incoming))
	for _, r := range existing {
		byQuote[r.Quote] = r
	}
	for _, r := range incoming {
		byQuote[r.Quote] = r
	}
	result := make([]Rate, 0, len(byQuote))
	for _, r := range byQuote {
		result = append(result, r)
	}
	return result
}
