package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
)

type mockAccountRepo struct {
	accounts map[uint]*entity.Account
	lastID   uint
}

func (m *mockAccountRepo) Create(account *entity.Account) error {
	m.lastID++
	account.ID = m.lastID
	if m.accounts == nil {
		m.accounts = make(map[uint]*entity.Account)
	}
	m.accounts[account.ID] = account
	return nil
}

func (m *mockAccountRepo) GetByID(id uint) (*entity.Account, error) {
	account, ok := m.accounts[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return account, nil
}

func (m *mockAccountRepo) GetByUserID(userID uint) ([]*entity.Account, error) {
	var list []*entity.Account
	for _, a := range m.accounts {
		if a.UserID == userID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *mockAccountRepo) Update(account *entity.Account) error {
	m.accounts[account.ID] = account
	return nil
}
func (m *mockAccountRepo) Delete(id uint) error { delete(m.accounts, id); return nil }
func (m *mockAccountRepo) GetByUserIDAndType(userID uint, accountType entity.AccountType) ([]*entity.Account, error) {
	return nil, nil
}
func (m *mockAccountRepo) UpdateBalance(id uint, newBalance float64) error  { return nil }
func (m *mockAccountRepo) AddToBalance(id uint, amount float64) error        { return nil }
func (m *mockAccountRepo) SubtractFromBalance(id uint, amount float64) error { return nil }
func (m *mockAccountRepo) HasTransactions(id uint) (bool, error)             { return false, nil }
func (m *mockAccountRepo) SetActive(id uint, active bool) error              { return nil }
func (m *mockAccountRepo) GetActiveAccounts(userID uint) ([]*entity.Account, error) {
	return nil, nil
}
func (m *mockAccountRepo) GetTotalBalance(userID uint) (float64, error) { return 0, nil }
func (m *mockAccountRepo) GetAccountsByTypeAndUser(userID uint, accountType entity.AccountType, activeOnly bool) ([]*entity.Account, error) {
	return nil, nil
}

type mockUserRepo struct {
	user *entity.User
	err  error
}

func (m *mockUserRepo) Create(user *entity.User) error                                          { return nil }
func (m *mockUserRepo) GetByID(id uint) (*entity.User, error)                                   { return m.user, m.err }
func (m *mockUserRepo) GetByEmail(email string) (*entity.User, error)                           { return nil, nil }
func (m *mockUserRepo) Update(user *entity.User) error                                          { return nil }
func (m *mockUserRepo) Delete(id uint) error                                                     { return nil }
func (m *mockUserRepo) UpdateLastLogin(id uint) error                                            { return nil }
func (m *mockUserRepo) SetActive(id uint, active bool) error                                     { return nil }
func (m *mockUserRepo) SetVerified(id uint, verified bool) error                                 { return nil }
func (m *mockUserRepo) GetActiveUsers() ([]*entity.User, error)                                 { return nil, nil }
func (m *mockUserRepo) CountUsers() (int64, error)                                               { return 0, nil }
func (m *mockUserRepo) GetUsersRegisteredAfter(date time.Time) ([]*entity.User, error)          { return nil, nil }

func TestAccountUseCase_validateAccountRules(t *testing.T) {
	uc := &AccountUseCase{}

	tests := []struct {
		name      string
		account   *entity.Account
		expectErr bool
	}{
		{
			name: "credit account requires positive limit",
			account: &entity.Account{
				Type:        entity.AccountTypeCredit,
				CreditLimit: 0,
			},
			expectErr: true,
		},
		{
			name: "non-credit account cannot start negative",
			account: &entity.Account{
				Type:           entity.AccountTypeSavings,
				InitialBalance: -1,
			},
			expectErr: true,
		},
		{
			name: "negative low-balance limit is invalid when alert enabled",
			account: &entity.Account{
				Type:            entity.AccountTypeChecking,
				LowBalanceAlert: true,
				LowBalanceLimit: -10,
			},
			expectErr: true,
		},
		{
			name: "valid checking account",
			account: &entity.Account{
				Type:            entity.AccountTypeChecking,
				InitialBalance:  100,
				LowBalanceAlert: true,
				LowBalanceLimit: 20,
			},
			expectErr: false,
		},
		{
			name: "valid credit account",
			account: &entity.Account{
				Type:        entity.AccountTypeCredit,
				CreditLimit: 500,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.validateAccountRules(tt.account)
			if tt.expectErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestAccountUseCase_Create_AppliesDefaults(t *testing.T) {
	accountRepo := &mockAccountRepo{accounts: map[uint]*entity.Account{}}
	userRepo := &mockUserRepo{
		user: &entity.User{ID: 99, IsActive: true, Currency: "COP"},
	}
	uc := NewAccountUseCase(accountRepo, userRepo)

	req := &dto.CreateAccountRequest{
		Name:           "Principal",
		Type:           entity.AccountTypeSavings,
		InitialBalance: 1500,
	}

	account, err := uc.Create(99, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.Currency != "COP" {
		t.Fatalf("expected currency COP from user profile, got %s", account.Currency)
	}
	if account.Color != "#007bff" {
		t.Fatalf("expected default color, got %s", account.Color)
	}
	if account.Balance != 1500 {
		t.Fatalf("expected balance to match initial balance, got %v", account.Balance)
	}
}

func TestAccountUseCase_TransferBalance_Success(t *testing.T) {
	accountRepo := &mockAccountRepo{
		accounts: map[uint]*entity.Account{
			1: {ID: 1, UserID: 7, Type: entity.AccountTypeChecking, Balance: 200, IsActive: true},
			2: {ID: 2, UserID: 7, Type: entity.AccountTypeSavings, Balance: 50, IsActive: true},
		},
	}
	userRepo := &mockUserRepo{user: &entity.User{ID: 7, IsActive: true}}
	uc := NewAccountUseCase(accountRepo, userRepo)

	if err := uc.TransferBalance(7, 1, 2, 100); err != nil {
		t.Fatalf("unexpected transfer error: %v", err)
	}

	if accountRepo.accounts[1].Balance != 100 {
		t.Fatalf("expected source balance 100, got %v", accountRepo.accounts[1].Balance)
	}
	if accountRepo.accounts[2].Balance != 150 {
		t.Fatalf("expected destination balance 150, got %v", accountRepo.accounts[2].Balance)
	}
}
