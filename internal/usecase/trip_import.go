package usecase

import (
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// TripImportUseCase sugiere y asigna gastos preexistentes al viaje en función
// de las fechas del propio viaje.
type TripImportUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	expenseRepo    repo.ExpenseRepo
	allocationRepo repo.TripBudgetAllocationRepo
}

// NewTripImportUseCase construye el use case de import
func NewTripImportUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	expenseRepo repo.ExpenseRepo,
	allocationRepo repo.TripBudgetAllocationRepo,
) *TripImportUseCase {
	return &TripImportUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		expenseRepo:    expenseRepo,
		allocationRepo: allocationRepo,
	}
}

// Suggest devuelve los gastos del usuario sin viaje asignado dentro del rango
func (uc *TripImportUseCase) Suggest(userID, tripID uint) ([]dto.ImportSuggestionResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	candidates, err := uc.expenseRepo.GetUnassignedTripCandidates(userID, trip.StartDate, trip.EndDate)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	out := make([]dto.ImportSuggestionResponse, 0, len(candidates))
	for _, expense := range candidates {
		out = append(out, dto.ImportSuggestionResponse{
			ExpenseID:   expense.ID,
			Amount:      expense.Amount,
			Currency:    expense.Currency,
			Description: expense.Description,
			Date:        expense.Date,
			CategoryID:  expense.CategoryID,
			Merchant:    expense.Merchant,
		})
	}
	return out, nil
}

// Assign asigna en lote gastos al viaje
func (uc *TripImportUseCase) Assign(userID, tripID uint, req *dto.AssignImportRequest) (int, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return 0, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return 0, ErrTripPermissionDenied
	}

	if err := uc.expenseRepo.AssignToTrip(req.ExpenseIDs, tripID, userID); err != nil {
		return 0, apperrors.ErrInternal.WithInternal(err)
	}
	return len(req.ExpenseIDs), nil
}

func (uc *TripImportUseCase) hasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	return err == nil && member != nil
}
