package entity

import "testing"

func TestAccount_CreditAccountFlow(t *testing.T) {
	account := &Account{
		Type:        AccountTypeCredit,
		IsActive:    true,
		CreditLimit: 1000,
		Balance:     200, // deuda actual
	}

	if !account.CanDebit(700) {
		t.Fatalf("expected debit to be allowed for available credit")
	}
	if ok := account.Debit(300); !ok {
		t.Fatalf("expected debit to succeed")
	}
	if account.Balance != 500 {
		t.Fatalf("expected balance 500 after debit, got %v", account.Balance)
	}

	account.Credit(800) // pago de deuda
	if account.Balance != 0 {
		t.Fatalf("expected debt floor at 0 after credit, got %v", account.Balance)
	}
}

func TestAccount_ShouldAlert(t *testing.T) {
	debit := &Account{
		Type:            AccountTypeSavings,
		IsActive:        true,
		LowBalanceAlert: true,
		LowBalanceLimit: 100,
		Balance:         80,
	}
	if !debit.ShouldAlert() {
		t.Fatalf("expected low-balance alert for savings account")
	}

	credit := &Account{
		Type:            AccountTypeCredit,
		IsActive:        true,
		CreditLimit:     1000,
		Balance:         950,
		LowBalanceAlert: true,
		LowBalanceLimit: 60, // disponible = 50 -> alerta
	}
	if !credit.ShouldAlert() {
		t.Fatalf("expected low-available-credit alert")
	}
}
