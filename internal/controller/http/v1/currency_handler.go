package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/pkg/currency"
	"github.com/nick130920/fintech-backend/pkg/exchange"
)

type CurrencyHandler struct {
	exchangeProvider exchange.Provider
}

type currenciesCatalogResponse struct {
	Data interface{} `json:"data"`
}

type exchangeRatesResponse struct {
	Base  string          `json:"base"`
	Rates []exchange.Rate `json:"rates"`
}

func NewCurrencyHandler(provider exchange.Provider) *CurrencyHandler {
	return &CurrencyHandler{exchangeProvider: provider}
}

// GetCurrencies godoc
// @Summary Listar monedas soportadas
// @Description Devuelve el catálogo de monedas soportadas por la aplicación
// @Tags currencies
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/currencies [get]
func (h *CurrencyHandler) GetCurrencies(c *gin.Context) {
	c.JSON(http.StatusOK, currenciesCatalogResponse{Data: currency.All()})
}

// GetExchangeRates godoc
// @Summary Obtener tipos de cambio
// @Description Obtiene tipos de cambio para una moneda base y símbolos opcionales
// @Tags currencies
// @Produce json
// @Param base query string false "Moneda base (default USD)"
// @Param symbols query string false "Lista de símbolos separados por coma (ej: EUR,MXN)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 502 {object} dto.ErrorResponse
// @Router /api/v1/currencies/rates [get]
func (h *CurrencyHandler) GetExchangeRates(c *gin.Context) {
	base := strings.ToUpper(c.DefaultQuery("base", "USD"))
	if !currency.IsValid(base) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "invalid base currency",
		})
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
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{
			Error:   "exchange_rates_failed",
			Message: "failed to fetch exchange rates: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, exchangeRatesResponse{Base: base, Rates: rates})
}
