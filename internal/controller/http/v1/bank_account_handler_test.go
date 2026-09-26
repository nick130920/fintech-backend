package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

type bankAccountRepoFake struct {
	repo.BankAccountRepo
	account            *entity.BankAccount
	setActiveCalls     int
	setActiveValue     bool
	updateBalanceCalls int
	updateBalanceValue float64
}

func (r *bankAccountRepoFake) GetByID(id uint) (*entity.BankAccount, error) {
	return r.account, nil
}

func (r *bankAccountRepoFake) SetActive(id uint, active bool) error {
	r.setActiveCalls++
	r.setActiveValue = active
	return nil
}

func (r *bankAccountRepoFake) UpdateBalance(id uint, balance float64) error {
	r.updateBalanceCalls++
	r.updateBalanceValue = balance
	return nil
}

func newBankAccountHandlerForTest(repository *bankAccountRepoFake) *BankAccountHandler {
	return NewBankAccountHandler(usecase.NewBankAccountUseCase(repository, nil), zerolog.Nop())
}

func TestBankAccountHandler_SetBankAccountActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		body            string
		wantStatus      int
		wantSetCalls    int
		wantActiveValue bool
	}{
		{
			name:            "explicit false updates account",
			body:            `{"is_active":false}`,
			wantStatus:      http.StatusNoContent,
			wantSetCalls:    1,
			wantActiveValue: false,
		},
		{
			name:         "missing active field is rejected without mutation",
			body:         `{}`,
			wantStatus:   http.StatusBadRequest,
			wantSetCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &bankAccountRepoFake{account: &entity.BankAccount{ID: 1, UserID: 1}}
			handler := newBankAccountHandlerForTest(repository)
			router := gin.New()
			router.PATCH("/bank-accounts/:id/active", func(c *gin.Context) {
				c.Set("user_id", uint(1))
				handler.SetBankAccountActive(c)
			})

			req := httptest.NewRequest(http.MethodPatch, "/bank-accounts/1/active", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if repository.setActiveCalls != tt.wantSetCalls {
				t.Fatalf("SetActive calls = %d, want %d", repository.setActiveCalls, tt.wantSetCalls)
			}
			if tt.wantSetCalls > 0 && repository.setActiveValue != tt.wantActiveValue {
				t.Fatalf("SetActive active = %t, want %t", repository.setActiveValue, tt.wantActiveValue)
			}
		})
	}
}

func TestBankAccountHandler_UpdateBankAccountBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		body             string
		wantStatus       int
		wantUpdateCalls  int
		wantBalanceValue float64
	}{
		{
			name:             "explicit zero updates balance",
			body:             `{"balance":0}`,
			wantStatus:       http.StatusNoContent,
			wantUpdateCalls:  1,
			wantBalanceValue: 0,
		},
		{
			name:            "missing balance field is rejected without mutation",
			body:            `{}`,
			wantStatus:      http.StatusBadRequest,
			wantUpdateCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &bankAccountRepoFake{account: &entity.BankAccount{ID: 1, UserID: 1}}
			handler := newBankAccountHandlerForTest(repository)
			router := gin.New()
			router.PATCH("/bank-accounts/:id/balance", func(c *gin.Context) {
				c.Set("user_id", uint(1))
				handler.UpdateBankAccountBalance(c)
			})

			req := httptest.NewRequest(http.MethodPatch, "/bank-accounts/1/balance", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if repository.updateBalanceCalls != tt.wantUpdateCalls {
				t.Fatalf("UpdateBalance calls = %d, want %d", repository.updateBalanceCalls, tt.wantUpdateCalls)
			}
			if tt.wantUpdateCalls > 0 && repository.updateBalanceValue != tt.wantBalanceValue {
				t.Fatalf("UpdateBalance balance = %v, want %v", repository.updateBalanceValue, tt.wantBalanceValue)
			}
		})
	}
}
