package usecase

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
)

// smsBudgetAnalysisExcluded: marketing, mora/facturas, recordatorios de pago, OTP, campañas bancarias.
// Se evalúa antes que cualquier inclusión para no mandar ruido a la IA.
var smsBudgetAnalysisExcluded = regexp.MustCompile(
	`(?i)(` +
		// Telco / servicios: cobranza y recordatorios (no movimiento contable confirmado)
		`\baviso\s+movistar\b|` +
		`movistar\s+te\s+informa.*\bmora\b|` +
		`tienes\s+\d+\s+productos?\s+en\s+mora|` +
		`por\s+mora\s+de|` +
		`proceso\s+jur[ií]dico|` +
		`si\s+pagaste\s+omite|` +
		`omite\s+(sms|este)|` +
		`haz\s+caso\s+omiso|` +
		`si\s+ya\s+cancel|` +
		`tu\s+factura\s+movistar|` +
		`\bfactura\s+de\s+tu\s+fijo\b|` +
		`\bfactura\s+movistar\b|` +
		`fijo\s+movistar|` +
		`\bvence\s+hoy\b|` +
		`HOY\s+VENCE|` +
		`evita\s+la\s+suspensi[oó]n|` +
		`proxim[oa]\s+a\s+vencer|` +
		`consulta\s+y\s+paga\s+tu\s+factura|` +
		`factura.*\best[aá]\s+lista\s+para\s+pago|` +
		`disponible\s+para\s+pago|` +
		`nunca\s+habia\s+sido\s+tan\s+facil\s+pagar\s+la\s+factura|` +
		`paga\s+facil\s+y\s+seguro|` +
		`#parapagos|` +
		`estar\s+al\s+dia\s+tu\s+mejor\s+referencia|` +
		// Marketing / promos / puntos
		`superoferta|hot\s+sale|` +
		`\bdcto\b|` +
		`\bdescuent|` +
		`\bTyC\b\s+https|` +
		`\bcup[oó]n\b|\bbono\b|` +
		`preaprob|` +
		`cmr\s+puntos|` +
		`\bpara\s+ganar\b|` +
		`\bcompra\s+ya\b|` +
		`exclusivo\s+en\s+app|` +
		`cambiate\s+a\s+claro|` +
		`portabilidad|` +
		`\b50GB\b|` +
		`midatacredito|midatacr[eé]dito|` +
		`\bplan\s+TOP\b|antifraude|` +
		`club\s+patprimo|redimelo\s+online|` +
		`\baddi:\s*¡|reto\s+del\s+mes|` +
		`multiplica\s+tu\s+cupo|` +
		`rappi:.*preventa|` +
		`#TuExito|` +
		`en\s+carulla\s+lleva|` +
		`en\s+#TuExito|` +
		`\btuya:\s*|` +
		`avanzo:.*beneficio|` +
		`sura:\s*¿que\s+tiene|` +
		`longevo|` +
		`manda\s+tu\s+coca|` +
		// Recordatorios de deuda / “pendiente” (no SMS de transacción ejecutada)
		`recordat[ea].{0,100}(pendiente|aun\s+esta)|` +
		`evita\s+cargos\s+adicionales|` +
		`probablemente\s+fue\s+un\s+descuido|` +
		`aun\s+esta\s+pendiente|` +
		`queremos\s+recordarte\s+que\s+tu\s+pago|` +
		// OTP / seguridad / apps
		`telegram\s+code|verification\s+code|your\s+code\s+is|` +
		`datos\s+que\s+ingresaste.*tarjeta|` +
		`google\s+pay.*incorrecto|` +
		`cuida\s+tus\s+datos` +
		// Campañas e insights Bancolombia (no movimiento)
		`|bancolombia:.{0,160}0%\s*interes|` +
		`bancol\.co/macsms|` +
		`sumarian\s+tus\s+gastos|` +
		`conocelos\s+en\s+app\s+mi\s+bancolombia|` +
		`dia\s+a\s+dia\s*>\s*gastos|` +
		// Otros
		`interrapidisimo.*descarga\s+la\s+factura|` +
		`icetex\s+sigue|` +
		`movistar\s+no\s+envia\s+mensajes|` +
		`f\.fcert\.co|` +
		`telefonicawebsites\.co|` +
		`recaudo\.epayco|` +
		`wa\.me/|` +
		`wa\.link/` +
		`)`,
)

// smsStrongBankMovement: confirmación explícita de compra, transferencia o pago ejecutado (Colombia + algunos patrones genéricos).
var smsStrongBankMovement = regexp.MustCompile(
	`(?i)(` +
		`bancolombia:\s*.{0,120}?\b(compraste|transferiste|recibiste\s+una\s+transferencia|pagaste)\b|` +
		`\btransferiste\s+\$|` +
		`\bcompraste\s+cop|` +
		`recibiste\s+una\s+transferencia\s+por|` +
		`\bpagaste\s+\$[\d,\.]+|` +
		`pagaste\s+.{0,40}\b(codigo\s+qr|desde\s+tu\s+cuenta)\b|` +
		`transferiste\s+.{0,30}boton\s+bancolombia|` +
		`nequi:\s*(te\s+envi|enviamos|recibiste)|` +
		`daviplata:.{0,80}?\b(transferiste|enviaste|recibiste|pagaste)\b|` +
		// México / otros: SPEI / CLABE con verbo de movimiento
		`spei\s+(recibido|enviado|entrante)|transferencia\s+recibida` +
		`)`,
)

// bankTransactionSMSRegexp: señales bancarias LATAM. Evitamos \bpago\b suelto (“Paga HOY”, “tu pago pendiente”).
var bankTransactionSMSRegexp = regexp.MustCompile(
	`(?i)(` +
		`[\$€£]|` +
		`\b(mxn|cop|clp|ars|brl|pen|usd|eur|gs\.|pyg|uyu|bob|crc|gtq|hnl|nio|dop|ves)\b|` +
		`compraste|\bcompra\b|consumo|pagaste|pago\s+realizado|pago\s+exitoso|comprobante\s+de\s+pago|realizaste\s+un\s+pago|` +
		`débito|debito|debit|cargo|abono|retiro|transferencia|transfer|` +
		`\bbanco\b|` +
		`spei|clabe|cbu|cvu|alias|pix|` +
		`bbva|banamex|santander|banorte|hsbc|scotiabank|inbursa|azteca|banregio|` +
		`bancolombia|nequi|daviplata|davivienda|banco\s+de\s+occidente|` +
		`interbank|bcp|continental|nubank|mercado.pago|rappi\s*bank|uala|brubank|` +
		`yape|plin|banco\s+nacion|banco\s+estado|` +
		`tarjeta|cuenta|ahorro|corriente|movimiento|transacci|` +
		`visa|mastercard|amex` +
		`)`,
)

// filterMessagesLikelyBankSMS deja solo SMS con alta probabilidad de ser movimiento real (no mora/marketing).
func filterMessagesLikelyBankSMS(messages []dto.SMSMessageForAnalysis) []dto.SMSMessageForAnalysis {
	var out []dto.SMSMessageForAnalysis
	for _, msg := range messages {
		if likelyBankTransactionSMS(msg.Body) {
			out = append(out, msg)
		}
	}
	return out
}

func likelyBankTransactionSMS(body string) bool {
	b := strings.TrimSpace(body)
	if utf8.RuneCountInString(b) < 12 {
		return false
	}
	if !strings.ContainsAny(b, "0123456789") {
		return false
	}
	if smsBudgetAnalysisExcluded.MatchString(b) {
		return false
	}
	if smsStrongBankMovement.MatchString(b) {
		return true
	}
	// Secundario: vocabulario bancario sin el ruido típico (ya excluido arriba). Sin fallback “texto largo”.
	return bankTransactionSMSRegexp.MatchString(b)
}
