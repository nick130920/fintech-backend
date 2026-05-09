package usecase

import (
	"sort"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"github.com/nick130920/fintech-backend/pkg/exchange"
)

// SettlementUseCase administra los pagos entre miembros que saldan deudas
type SettlementUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	splitRepo      repo.ExpenseSplitRepo
	settlementRepo repo.SettlementRepo
	expenseRepo    repo.ExpenseRepo
	fxProvider     exchange.Provider
}

// NewSettlementUseCase construye el use case
func NewSettlementUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	splitRepo repo.ExpenseSplitRepo,
	settlementRepo repo.SettlementRepo,
	expenseRepo repo.ExpenseRepo,
	fxProvider exchange.Provider,
) *SettlementUseCase {
	return &SettlementUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		splitRepo:      splitRepo,
		settlementRepo: settlementRepo,
		expenseRepo:    expenseRepo,
		fxProvider:     fxProvider,
	}
}

// List devuelve los settlements del viaje
func (uc *SettlementUseCase) List(userID, tripID uint) ([]*dto.SettlementResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	settlements, err := uc.settlementRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := make([]*dto.SettlementResponse, 0, len(settlements))
	for _, s := range settlements {
		out = append(out, mapSettlement(s))
	}
	return out, nil
}

// Create registra un pago entre miembros y marca splits pendientes como pagados
func (uc *SettlementUseCase) Create(userID, tripID uint, req *dto.CreateSettlementRequest) (*dto.SettlementResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	from, err := uc.memberRepo.GetByTripAndID(tripID, req.FromMemberID)
	if err != nil || from == nil {
		return nil, ErrTripMemberNotFound
	}
	to, err := uc.memberRepo.GetByTripAndID(tripID, req.ToMemberID)
	if err != nil || to == nil {
		return nil, ErrTripMemberNotFound
	}

	currency := req.Currency
	if currency == "" {
		currency = trip.PrimaryCurrency
	}
	fxRate := req.FxRate
	if fxRate <= 0 {
		fxRate = uc.resolveFxRate(currency, trip.PrimaryCurrency)
	}

	paidAt := req.PaidAt
	if paidAt.IsZero() {
		paidAt = time.Now()
	}

	settlement := &entity.Settlement{
		TripID:         tripID,
		FromMemberID:   req.FromMemberID,
		ToMemberID:     req.ToMemberID,
		RecordedByUser: userID,
		Amount:         req.Amount,
		Currency:       currency,
		FxRate:         fxRate,
		PaidAt:         paidAt,
		Notes:          req.Notes,
	}

	if err := uc.settlementRepo.Create(settlement); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	// Marcar splits del pagador hacia el receptor en orden FIFO hasta cubrir el monto
	uc.markSplitsAsPaid(tripID, req.FromMemberID, req.ToMemberID, req.Amount, currency, trip.PrimaryCurrency, fxRate)

	return mapSettlement(settlement), nil
}

// Delete elimina un settlement (revierte el saldo lógicamente al recalcular balance)
func (uc *SettlementUseCase) Delete(userID, tripID, settlementID uint) error {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return ErrTripPermissionDenied
	}
	return uc.settlementRepo.Delete(settlementID)
}

// markSplitsAsPaid recorre los splits del pagador hacia el receptor y los marca
// como pagados hasta consumir el monto del settlement (en moneda primaria).
func (uc *SettlementUseCase) markSplitsAsPaid(tripID, fromMemberID, toMemberID uint, amount float64, currency, primary string, fxRate float64) {
	amountInPrimary := amount
	if currency != primary && fxRate > 0 {
		amountInPrimary = amount * fxRate
	}

	expenses, err := uc.expenseRepo.GetByTripID(tripID)
	if err != nil {
		return
	}

	type splitOrder struct {
		split   *entity.ExpenseSplit
		expense *entity.Expense
	}
	var pendingSplits []splitOrder
	for _, expense := range expenses {
		if expense.PaidByMemberID == nil || *expense.PaidByMemberID != toMemberID {
			continue
		}
		for i := range expense.Splits {
			split := &expense.Splits[i]
			if split.MemberID == fromMemberID && !split.IsPaid {
				pendingSplits = append(pendingSplits, splitOrder{split: split, expense: expense})
			}
		}
	}

	sort.Slice(pendingSplits, func(i, j int) bool {
		return pendingSplits[i].expense.Date.Before(pendingSplits[j].expense.Date)
	})

	remaining := amountInPrimary
	for _, item := range pendingSplits {
		share := convertSplit(*item.split, item.expense, primary)
		if remaining < share-0.01 {
			break
		}
		item.split.MarkPaid()
		_ = uc.splitRepo.Update(item.split)
		remaining -= share
	}
}

func (uc *SettlementUseCase) hasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	return err == nil && member != nil
}

func (uc *SettlementUseCase) resolveFxRate(from, to string) float64 {
	if from == to || uc.fxProvider == nil {
		return 1
	}
	rate, err := uc.fxProvider.GetRate(from, to)
	if err != nil || rate == nil {
		return 1
	}
	return rate.Rate
}

func mapSettlement(s *entity.Settlement) *dto.SettlementResponse {
	out := &dto.SettlementResponse{
		ID:           s.ID,
		TripID:       s.TripID,
		FromMemberID: s.FromMemberID,
		ToMemberID:   s.ToMemberID,
		Amount:       s.Amount,
		Currency:     s.Currency,
		FxRate:       s.FxRate,
		PaidAt:       s.PaidAt,
		Notes:        s.Notes,
		CreatedAt:    s.CreatedAt,
	}
	if s.FromMember != nil {
		out.FromName = s.FromMember.DisplayName
	}
	if s.ToMember != nil {
		out.ToName = s.ToMember.DisplayName
	}
	return out
}
