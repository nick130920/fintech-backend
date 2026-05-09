package usecase

import (
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"github.com/nick130920/fintech-backend/pkg/finance/debtsimplify"
)

// TripBalanceUseCase calcula los saldos netos por miembro y propone
// la lista mínima de transferencias para saldarlos.
type TripBalanceUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	expenseRepo    repo.ExpenseRepo
	splitRepo      repo.ExpenseSplitRepo
	settlementRepo repo.SettlementRepo
}

// NewTripBalanceUseCase construye el use case de balance
func NewTripBalanceUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	expenseRepo repo.ExpenseRepo,
	splitRepo repo.ExpenseSplitRepo,
	settlementRepo repo.SettlementRepo,
) *TripBalanceUseCase {
	return &TripBalanceUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		expenseRepo:    expenseRepo,
		splitRepo:      splitRepo,
		settlementRepo: settlementRepo,
	}
}

// Compute calcula el balance del viaje
func (uc *TripBalanceUseCase) Compute(userID, tripID uint) (*dto.TripBalanceResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	members, err := uc.memberRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	memberByID := make(map[uint]*entity.TripMember, len(members))
	balances := make(map[uint]float64, len(members))
	for _, m := range members {
		memberByID[m.ID] = m
		balances[m.ID] = 0
	}

	expenses, err := uc.expenseRepo.GetByTripID(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	for _, expense := range expenses {
		amount := convertToPrimary(expense, trip.PrimaryCurrency)
		if expense.PaidByMemberID != nil {
			balances[*expense.PaidByMemberID] += amount
		}
		// El monto a cargo de cada miembro se descuenta de su balance
		for _, split := range expense.Splits {
			balances[split.MemberID] -= convertSplit(split, expense, trip.PrimaryCurrency)
		}
	}

	// Restar settlements ya realizados (un settlement reduce la deuda del pagador)
	settlements, err := uc.settlementRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	for _, s := range settlements {
		amount := s.Amount
		if s.Currency != trip.PrimaryCurrency && s.FxRate > 0 {
			amount = s.Amount * s.FxRate
		}
		balances[s.FromMemberID] += amount
		balances[s.ToMemberID] -= amount
	}

	// Construir respuesta de balances por miembro
	netByMember := make([]dto.TripMemberBalance, 0, len(members))
	for _, m := range members {
		netByMember = append(netByMember, dto.TripMemberBalance{
			MemberID:   m.ID,
			MemberName: m.DisplayName,
			NetAmount:  round2Plain(balances[m.ID]),
		})
	}

	transfers := debtsimplify.Simplify(balances)
	out := make([]dto.TripBalanceTransfer, 0, len(transfers))
	for _, t := range transfers {
		entry := dto.TripBalanceTransfer{
			FromMemberID: t.From,
			ToMemberID:   t.To,
			Amount:       t.Amount,
		}
		if member, ok := memberByID[t.From]; ok {
			entry.FromName = member.DisplayName
		}
		if member, ok := memberByID[t.To]; ok {
			entry.ToName = member.DisplayName
		}
		out = append(out, entry)
	}

	return &dto.TripBalanceResponse{
		TripID:      tripID,
		Currency:    trip.PrimaryCurrency,
		NetByMember: netByMember,
		Transfers:   out,
	}, nil
}

func (uc *TripBalanceUseCase) hasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	return err == nil && member != nil
}

func convertToPrimary(expense *entity.Expense, primary string) float64 {
	if expense.Currency == primary || expense.ExchangeRate <= 0 {
		return expense.Amount
	}
	return expense.Amount * expense.ExchangeRate
}

func convertSplit(split entity.ExpenseSplit, expense *entity.Expense, primary string) float64 {
	if expense.Currency == primary || expense.ExchangeRate <= 0 {
		return split.ShareAmount
	}
	return split.ShareAmount * expense.ExchangeRate
}

func round2Plain(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
