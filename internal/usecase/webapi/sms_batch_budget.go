package webapi

// BatchSMSBudgetLine is one line from a numbered SMS block that the model classified.
type BatchSMSBudgetLine struct {
	Line            int     `json:"line"`
	Amount          float64 `json:"amount"`
	TransactionType string  `json:"transaction_type"` // expense, income, ignore
	Confidence      float64 `json:"confidence"`
}

// BatchSMSBudgetResponse is the JSON shape returned by batch budget extraction.
type BatchSMSBudgetResponse struct {
	Lines []BatchSMSBudgetLine `json:"lines"`
}
