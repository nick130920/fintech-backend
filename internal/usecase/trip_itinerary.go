package usecase

import (
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// TripItineraryUseCase administra los items del itinerario del viaje
type TripItineraryUseCase struct {
	tripRepo      repo.TripRepo
	memberRepo    repo.TripMemberRepo
	itineraryRepo repo.TripItineraryRepo
	expenseRepo   repo.ExpenseRepo
}

// NewTripItineraryUseCase construye el use case
func NewTripItineraryUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	itineraryRepo repo.TripItineraryRepo,
	expenseRepo repo.ExpenseRepo,
) *TripItineraryUseCase {
	return &TripItineraryUseCase{
		tripRepo:      tripRepo,
		memberRepo:    memberRepo,
		itineraryRepo: itineraryRepo,
		expenseRepo:   expenseRepo,
	}
}

// List devuelve los items del itinerario
func (uc *TripItineraryUseCase) List(userID, tripID uint) ([]dto.TripItineraryResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	items, err := uc.itineraryRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	out := make([]dto.TripItineraryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, mapItinerary(item))
	}
	return out, nil
}

// Create agrega un item al itinerario
func (uc *TripItineraryUseCase) Create(userID, tripID uint, req *dto.CreateItineraryItemRequest) (*dto.TripItineraryResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if err := uc.assertCanManage(trip, userID); err != nil {
		return nil, err
	}

	currency := req.Currency
	if currency == "" {
		currency = trip.PrimaryCurrency
	}

	item := &entity.TripItineraryItem{
		TripID:        tripID,
		Day:           req.Day,
		Time:          req.Time,
		Type:          entity.TripItineraryType(req.Type),
		Title:         req.Title,
		Description:   req.Description,
		Location:      req.Location,
		EstimatedCost: req.EstimatedCost,
		Currency:      currency,
	}
	if err := uc.itineraryRepo.Create(item); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := mapItinerary(item)
	return &out, nil
}

// Update modifica un item del itinerario
func (uc *TripItineraryUseCase) Update(userID, tripID, itemID uint, req *dto.UpdateItineraryItemRequest) (*dto.TripItineraryResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if err := uc.assertCanManage(trip, userID); err != nil {
		return nil, err
	}

	item, err := uc.itineraryRepo.GetByID(itemID)
	if err != nil || item == nil || item.TripID != tripID {
		return nil, apperrors.ErrNotFound.WithDetails("item de itinerario no encontrado")
	}

	if req.Day != nil {
		item.Day = *req.Day
	}
	if req.Time != nil {
		item.Time = *req.Time
	}
	if req.Type != nil {
		item.Type = entity.TripItineraryType(*req.Type)
	}
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Location != nil {
		item.Location = *req.Location
	}
	if req.EstimatedCost != nil {
		item.EstimatedCost = *req.EstimatedCost
	}
	if req.Currency != nil {
		item.Currency = *req.Currency
	}

	if err := uc.itineraryRepo.Update(item); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := mapItinerary(item)
	return &out, nil
}

// Delete elimina un item del itinerario
func (uc *TripItineraryUseCase) Delete(userID, tripID, itemID uint) error {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return ErrTripNotFound
	}
	if err := uc.assertCanManage(trip, userID); err != nil {
		return err
	}
	item, err := uc.itineraryRepo.GetByID(itemID)
	if err != nil || item == nil || item.TripID != tripID {
		return apperrors.ErrNotFound.WithDetails("item de itinerario no encontrado")
	}
	return uc.itineraryRepo.Delete(itemID)
}

// LinkExpense vincula un gasto real al item del itinerario
func (uc *TripItineraryUseCase) LinkExpense(userID, tripID, itemID, expenseID uint) (*dto.TripItineraryResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if err := uc.assertCanManage(trip, userID); err != nil {
		return nil, err
	}

	item, err := uc.itineraryRepo.GetByID(itemID)
	if err != nil || item == nil || item.TripID != tripID {
		return nil, apperrors.ErrNotFound.WithDetails("item de itinerario no encontrado")
	}
	expense, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil || expense == nil {
		return nil, apperrors.ErrExpenseNotFound
	}
	if expense.TripID == nil || *expense.TripID != tripID {
		return nil, apperrors.ErrInvalidRequest.WithDetails("el gasto no pertenece a este viaje")
	}

	if err := uc.itineraryRepo.LinkExpense(itemID, expenseID); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	updated, err := uc.itineraryRepo.GetByID(itemID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := mapItinerary(updated)
	return &out, nil
}

func (uc *TripItineraryUseCase) hasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	return err == nil && member != nil
}

func (uc *TripItineraryUseCase) assertCanManage(trip *entity.Trip, userID uint) error {
	if trip.OwnerUserID == userID {
		return nil
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	if err != nil || member == nil || !member.CanManage() {
		return ErrTripPermissionDenied
	}
	return nil
}
