package usecase

import (
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// TripBudgetUseCase administra las asignaciones de presupuesto del viaje
type TripBudgetUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	allocationRepo repo.TripBudgetAllocationRepo
	categoryRepo   repo.CategoryRepo
}

// NewTripBudgetUseCase construye el use case
func NewTripBudgetUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	allocationRepo repo.TripBudgetAllocationRepo,
	categoryRepo repo.CategoryRepo,
) *TripBudgetUseCase {
	return &TripBudgetUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		allocationRepo: allocationRepo,
		categoryRepo:   categoryRepo,
	}
}

// GetAllocations devuelve las asignaciones del presupuesto de un viaje
func (uc *TripBudgetUseCase) GetAllocations(userID, tripID uint) ([]dto.TripAllocationResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	allocations, err := uc.allocationRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := make([]dto.TripAllocationResponse, 0, len(allocations))
	for _, a := range allocations {
		out = append(out, mapAllocation(a, trip.DaysRemaining()))
	}
	return out, nil
}

// UpsertBudget reemplaza/actualiza las asignaciones del presupuesto del viaje
func (uc *TripBudgetUseCase) UpsertBudget(userID, tripID uint, req *dto.UpsertTripBudgetRequest) ([]dto.TripAllocationResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if err := uc.assertCanManage(trip, userID); err != nil {
		return nil, err
	}

	totalEstimated := 0.0
	for _, item := range req.Allocations {
		category, errCat := uc.categoryRepo.GetByID(item.CategoryID)
		if errCat != nil || category == nil {
			return nil, apperrors.ErrCategoryNotFound
		}
		currency := item.Currency
		if currency == "" {
			currency = trip.PrimaryCurrency
		}

		allocation := &entity.TripBudgetAllocation{
			TripID:          tripID,
			CategoryID:      item.CategoryID,
			EstimatedAmount: item.EstimatedAmount,
			Currency:        currency,
			Notes:           item.Notes,
		}
		if err := uc.allocationRepo.Upsert(allocation); err != nil {
			return nil, apperrors.ErrInternal.WithInternal(err)
		}
		totalEstimated += item.EstimatedAmount
	}

	// Actualizar total estimado del viaje
	if err := uc.tripRepo.UpdateTotals(tripID, totalEstimated, trip.SpentTotal); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	allocations, err := uc.allocationRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := make([]dto.TripAllocationResponse, 0, len(allocations))
	for _, a := range allocations {
		out = append(out, mapAllocation(a, trip.DaysRemaining()))
	}
	return out, nil
}

func (uc *TripBudgetUseCase) hasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	return err == nil && member != nil
}

func (uc *TripBudgetUseCase) assertCanManage(trip *entity.Trip, userID uint) error {
	if trip.OwnerUserID == userID {
		return nil
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	if err != nil || member == nil || !member.CanManage() {
		return ErrTripPermissionDenied
	}
	return nil
}
