package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/pkg/currency"
	"github.com/nick130920/fintech-backend/pkg/exchange"
)

type CurrencyHandler struct {
	exchangeProvider exchange.Provider
}

func NewCurrencyHandler(provider exchange.Provider) *CurrencyHandler {
	return &CurrencyHandler{exchangeProvider: provider}
}

func (h *CurrencyHandler) GetCurrencies(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": currency.All(),
	})
}

func (h *CurrencyHandler) GetExchangeRates(c *gin.Context) {
	base := strings.ToUpper(c.DefaultQuery("base", "USD"))
	if !currency.IsValid(base) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base currency"})
		return
	}

	symbolsRaw := c.Query("symbols")
	var quotes []string
	if symbolsRaw != "" {
		for _, s := range strings.Split(symbolsRaw, ",") {
			s = strings.TrimSpace(strings.ToUpper(s))
			if s != "" && currency.IsValid(s) {
				quotes = append(quotes, s)
			}
		}
	}

	rates, err := h.exchangeProvider.GetRates(base, quotes)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch exchange rates", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"base":  base,
		"rates": rates,
	})
}
