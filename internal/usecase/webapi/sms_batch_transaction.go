package webapi

// BatchSMSTransactionLine is one numbered SMS line from a batch extraction (crear transacciones).
type BatchSMSTransactionLine struct {
	Line            int     `json:"line"`
	Success         bool    `json:"success"`
	Amount          float64 `json:"amount"`
	Description     string  `json:"description"`
	Merchant        string  `json:"merchant"`
	Date            string  `json:"date"`
	TransactionType string  `json:"transaction_type"` // expense, income, transfer
	Confidence      float64 `json:"confidence"`
	Currency        string  `json:"currency"`
}

// BatchSMSTransactionResponse is the JSON shape from IA for process-sms-batch.
type BatchSMSTransactionResponse struct {
	Lines []BatchSMSTransactionLine `json:"lines"`
}
