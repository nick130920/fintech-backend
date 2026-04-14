package v1

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// countryToCurrency mapea código ISO de país a código de moneda.
var countryToCurrency = map[string]string{
	"AR": "ARS", "BO": "BOB", "BR": "BRL", "CL": "CLP", "CO": "COP",
	"CR": "CRC", "CU": "CUP", "DO": "DOP", "EC": "USD", "SV": "USD",
	"GT": "GTQ", "HN": "HNL", "MX": "MXN", "NI": "NIO", "PA": "PAB",
	"PY": "PYG", "PE": "PEN", "UY": "UYU", "VE": "VES",
	"US": "USD", "CA": "CAD", "GB": "GBP", "AU": "AUD", "NZ": "NZD",
	"JP": "JPY", "CN": "CNY", "IN": "INR", "KR": "KRW", "SG": "SGD",
	"HK": "HKD", "CH": "CHF", "SE": "SEK", "NO": "NOK", "DK": "DKK",
	"DE": "EUR", "FR": "EUR", "ES": "EUR", "IT": "EUR", "PT": "EUR",
	"NL": "EUR", "BE": "EUR", "AT": "EUR", "FI": "EUR", "GR": "EUR",
	"ZA": "ZAR", "NG": "NGN", "EG": "EGP", "KE": "KES", "GH": "GHS",
	"RU": "RUB", "PL": "PLN", "CZ": "CZK", "HU": "HUF", "RO": "RON",
	"TR": "TRY", "SA": "SAR", "AE": "AED", "IL": "ILS", "PH": "PHP",
	"TH": "THB", "ID": "IDR", "MY": "MYR", "VN": "VND", "PK": "PKR",
}

type geoResponse struct {
	CountryCode  string `json:"country_code"`
	CurrencyCode string `json:"currency_code"`
}

// ipAPIResponse es la respuesta de ip-api.com
type ipAPIResponse struct {
	CountryCode string `json:"countryCode"`
	Status      string `json:"status"`
}

var httpClient = &http.Client{Timeout: 3 * time.Second}

// GetCountryFromIP detecta el país del cliente por IP y devuelve la moneda sugerida.
// Prioridad: 1) CF-IPCountry header (Cloudflare/Railway), 2) ip-api.com, 3) USD por defecto.
func GetCountryFromIP(c *gin.Context) {
	countryCode := ""

	// 1. Cloudflare/Railway inyecta este header automáticamente
	if cf := c.GetHeader("CF-IPCountry"); cf != "" && cf != "XX" {
		countryCode = strings.ToUpper(cf)
	}

	// 2. Fallback: consultar ip-api.com (gratis, sin API key, 45 req/min por IP)
	if countryCode == "" {
		clientIP := c.ClientIP()
		if clientIP != "" && clientIP != "::1" && clientIP != "127.0.0.1" {
			if resp, err := httpClient.Get("http://ip-api.com/json/" + clientIP + "?fields=countryCode,status"); err == nil {
				defer resp.Body.Close()
				var ipData ipAPIResponse
				if json.NewDecoder(resp.Body).Decode(&ipData) == nil && ipData.Status == "success" {
					countryCode = ipData.CountryCode
				}
			}
		}
	}

	currencyCode := "USD"
	if countryCode != "" {
		if c, ok := countryToCurrency[countryCode]; ok {
			currencyCode = c
		}
	}

	c.JSON(http.StatusOK, geoResponse{
		CountryCode:  countryCode,
		CurrencyCode: currencyCode,
	})
}
