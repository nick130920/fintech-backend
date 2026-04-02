package entity

import "testing"

func TestTransaction_SetAndGetTags(t *testing.T) {
	tx := &Transaction{}
	if err := tx.SetTags([]string{"hogar", "super"}); err != nil {
		t.Fatalf("unexpected error setting tags: %v", err)
	}

	got := tx.GetTags()
	if len(got) != 2 || got[0] != "hogar" || got[1] != "super" {
		t.Fatalf("unexpected tags: %#v", got)
	}
}

func TestTransaction_GetSignedAmountForAccount(t *testing.T) {
	toAccount := uint(20)
	tx := &Transaction{
		Type:        TransactionTypeTransfer,
		AccountID:   10,
		ToAccountID: &toAccount,
		Amount:      150,
	}

	if v := tx.GetSignedAmountForAccount(10); v != -150 {
		t.Fatalf("expected -150 for source account, got %v", v)
	}
	if v := tx.GetSignedAmountForAccount(20); v != 150 {
		t.Fatalf("expected 150 for destination account, got %v", v)
	}
	if v := tx.GetSignedAmountForAccount(30); v != 0 {
		t.Fatalf("expected 0 for unrelated account, got %v", v)
	}
}

func TestTransaction_SetAIConfidenceAndNeedsReview(t *testing.T) {
	tx := &Transaction{
		Source: TransactionSourceNotification,
		Status: TransactionStatusPending,
	}

	tx.SetAIConfidence(0.95)
	if tx.ValidationStatus != ValidationStatusAuto || tx.Status != TransactionStatusCompleted {
		t.Fatalf("expected auto/completed for high confidence, got %s/%s", tx.ValidationStatus, tx.Status)
	}

	tx.Status = TransactionStatusPending
	tx.SetAIConfidence(0.6)
	if tx.ValidationStatus != ValidationStatusPending {
		t.Fatalf("expected pending review for low confidence, got %s", tx.ValidationStatus)
	}
	if !tx.NeedsReview() {
		t.Fatalf("expected transaction to need review")
	}
}
