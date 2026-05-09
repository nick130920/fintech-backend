package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"github.com/nick130920/fintech-backend/pkg/exchange"
	"github.com/nick130920/fintech-backend/pkg/logger"
	"github.com/rs/zerolog"
)

// Errores específicos del módulo de viajes
var (
	ErrTripNotFound          = apperrors.NewAppError("TRIP_NOT_FOUND", "Viaje no encontrado", http.StatusNotFound)
	ErrTripPermissionDenied  = apperrors.NewAppError("TRIP_FORBIDDEN", "No tienes acceso a este viaje", http.StatusForbidden)
	ErrTripAlreadyCancelled  = apperrors.NewAppError("TRIP_ALREADY_CANCELLED", "El viaje ya fue cancelado", http.StatusConflict)
	ErrTripAlreadyCompleted  = apperrors.NewAppError("TRIP_ALREADY_COMPLETED", "El viaje ya fue completado", http.StatusConflict)
	ErrTripCannotStart       = apperrors.NewAppError("TRIP_CANNOT_START", "El viaje no puede iniciarse en su estado actual", http.StatusConflict)
	ErrTripMemberNotFound    = apperrors.NewAppError("TRIP_MEMBER_NOT_FOUND", "Miembro del viaje no encontrado", http.StatusNotFound)
	ErrTripMemberHasDebts    = apperrors.NewAppError("TRIP_MEMBER_HAS_DEBTS", "El miembro tiene deudas pendientes", http.StatusConflict)
	ErrTripInvitationInvalid = apperrors.NewAppError("TRIP_INVITATION_INVALID", "Invitación inválida o expirada", http.StatusBadRequest)
	ErrTripExpenseNotFound   = apperrors.NewAppError("TRIP_EXPENSE_NOT_FOUND", "Gasto del viaje no encontrado", http.StatusNotFound)
	ErrTripSplitsMismatch    = apperrors.NewAppError("TRIP_SPLITS_MISMATCH", "La suma de los splits no coincide con el monto del gasto", http.StatusBadRequest)
)

// TripUseCase agrupa la lógica para CRUD del viaje y transiciones de estado
type TripUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	allocationRepo repo.TripBudgetAllocationRepo
	itineraryRepo  repo.TripItineraryRepo
	userRepo       repo.UserRepo
	fxProvider     exchange.Provider
	logger         zerolog.Logger
}

// NewTripUseCase construye el use case de viajes
func NewTripUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	allocationRepo repo.TripBudgetAllocationRepo,
	itineraryRepo repo.TripItineraryRepo,
	userRepo repo.UserRepo,
	fxProvider exchange.Provider,
) *TripUseCase {
	return &TripUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		allocationRepo: allocationRepo,
		itineraryRepo:  itineraryRepo,
		userRepo:       userRepo,
		fxProvider:     fxProvider,
		logger:         logger.Get().With().Str("usecase", "Trip").Logger(),
	}
}

// CreateTrip crea un viaje y agrega al owner como primer miembro
func (uc *TripUseCase) CreateTrip(userID uint, req *dto.CreateTripRequest) (*dto.TripResponse, error) {
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	primaryCurrency := req.PrimaryCurrency
	if primaryCurrency == "" {
		primaryCurrency = user.Currency
	}

	fxRate := 1.0
	if uc.fxProvider != nil && primaryCurrency != user.Currency {
		if rate, errRate := uc.fxProvider.GetRate(primaryCurrency, user.Currency); errRate == nil && rate != nil {
			fxRate = rate.Rate
		}
	}

	trip := &entity.Trip{
		OwnerUserID:            userID,
		Name:                   req.Name,
		Destination:            req.Destination,
		CountryCode:            req.CountryCode,
		StartDate:              req.StartDate,
		EndDate:                req.EndDate,
		PrimaryCurrency:        primaryCurrency,
		BaseCurrencyAtCreation: user.Currency,
		FxRateAtCreation:       fxRate,
		Status:                 entity.TripStatusPlanning,
		CoverImageURL:          req.CoverImageURL,
		Notes:                  req.Notes,
	}

	if err := uc.tripRepo.Create(trip); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	owner := &entity.TripMember{
		TripID:      trip.ID,
		UserID:      &userID,
		DisplayName: user.FullName(),
		Role:        entity.TripMemberRoleOwner,
		IsGhost:     false,
		JoinedAt:    time.Now(),
	}
	if user.Email != "" {
		owner.Email = &user.Email
	}
	if err := uc.memberRepo.Create(owner); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	deepTrip, err := uc.tripRepo.GetByIDDeep(trip.ID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	return uc.buildTripResponse(deepTrip), nil
}

// GetTrip obtiene un viaje verificando acceso del usuario
func (uc *TripUseCase) GetTrip(userID, tripID uint) (*dto.TripResponse, error) {
	trip, err := uc.tripRepo.GetByIDDeep(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.userHasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}
	return uc.buildTripResponse(trip), nil
}

// ListTrips lista los viajes donde el usuario participa
func (uc *TripUseCase) ListTrips(userID uint, status string) ([]*dto.TripResponse, error) {
	trips, err := uc.tripRepo.GetByUser(userID, status)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	result := make([]*dto.TripResponse, 0, len(trips))
	for _, trip := range trips {
		full, errDeep := uc.tripRepo.GetByIDDeep(trip.ID)
		if errDeep != nil {
			continue
		}
		result = append(result, uc.buildTripResponse(full))
	}
	return result, nil
}

// UpdateTrip aplica cambios parciales sobre un viaje
func (uc *TripUseCase) UpdateTrip(userID, tripID uint, req *dto.UpdateTripRequest) (*dto.TripResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if err := uc.assertCanManage(trip, userID); err != nil {
		return nil, err
	}

	if req.Name != nil {
		trip.Name = *req.Name
	}
	if req.Destination != nil {
		trip.Destination = *req.Destination
	}
	if req.CountryCode != nil {
		trip.CountryCode = *req.CountryCode
	}
	if req.StartDate != nil {
		trip.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		trip.EndDate = *req.EndDate
	}
	if req.PrimaryCurrency != nil {
		trip.PrimaryCurrency = *req.PrimaryCurrency
	}
	if req.CoverImageURL != nil {
		trip.CoverImageURL = *req.CoverImageURL
	}
	if req.Notes != nil {
		trip.Notes = *req.Notes
	}

	if err := uc.tripRepo.Update(trip); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	deep, err := uc.tripRepo.GetByIDDeep(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	return uc.buildTripResponse(deep), nil
}

// DeleteTrip realiza un borrado lógico del viaje
func (uc *TripUseCase) DeleteTrip(userID, tripID uint) error {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return ErrTripNotFound
	}
	if trip.OwnerUserID != userID {
		return ErrTripPermissionDenied
	}
	return uc.tripRepo.Delete(tripID)
}

// ChangeStatus aplica una transición de estado validando reglas
func (uc *TripUseCase) ChangeStatus(userID, tripID uint, status entity.TripStatus) (*dto.TripResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if err := uc.assertCanManage(trip, userID); err != nil {
		return nil, err
	}

	switch status {
	case entity.TripStatusActive:
		if !trip.CanBeStarted() {
			return nil, ErrTripCannotStart
		}
	case entity.TripStatusCompleted:
		if !trip.CanBeCompleted() {
			return nil, ErrTripAlreadyCompleted
		}
	case entity.TripStatusCancelled:
		if !trip.CanBeCancelled() {
			return nil, ErrTripAlreadyCancelled
		}
	default:
		return nil, apperrors.ErrInvalidRequest.WithDetails("estado inválido")
	}

	trip.Status = status
	if err := uc.tripRepo.Update(trip); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	deep, err := uc.tripRepo.GetByIDDeep(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	return uc.buildTripResponse(deep), nil
}

// ----- Helpers compartidos -----

// userHasAccess valida si el usuario es owner o miembro del viaje
func (uc *TripUseCase) userHasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	for _, m := range trip.Members {
		if m.MatchesUser(userID) {
			return true
		}
	}
	if member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID); err == nil && member != nil {
		return true
	}
	return false
}

// assertCanManage devuelve error si el usuario no puede modificar el viaje
func (uc *TripUseCase) assertCanManage(trip *entity.Trip, userID uint) error {
	if trip.OwnerUserID == userID {
		return nil
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	if err != nil || member == nil {
		return ErrTripPermissionDenied
	}
	if !member.CanManage() {
		return ErrTripPermissionDenied
	}
	return nil
}

// buildTripResponse arma la respuesta detallada del viaje
func (uc *TripUseCase) buildTripResponse(trip *entity.Trip) *dto.TripResponse {
	resp := &dto.TripResponse{
		ID:              trip.ID,
		OwnerUserID:     trip.OwnerUserID,
		Name:            trip.Name,
		Destination:     trip.Destination,
		CountryCode:     trip.CountryCode,
		StartDate:       trip.StartDate,
		EndDate:         trip.EndDate,
		PrimaryCurrency: trip.PrimaryCurrency,
		Status:          string(trip.Status),
		CoverImageURL:   trip.CoverImageURL,
		EstimatedTotal:  trip.EstimatedTotal,
		SpentTotal:      trip.SpentTotal,
		RemainingAmount: trip.RemainingAmount(),
		ProgressPercent: trip.ProgressPercentage(),
		DaysTotal:       trip.DaysTotal(),
		DaysRemaining:   trip.DaysRemaining(),
		IsActiveNow:     trip.IsActiveNow(),
		Notes:           trip.Notes,
		CreatedAt:       trip.CreatedAt,
		UpdatedAt:       trip.UpdatedAt,
	}

	resp.Members = make([]dto.TripMemberResponse, 0, len(trip.Members))
	for _, m := range trip.Members {
		resp.Members = append(resp.Members, mapMember(&m))
	}

	resp.Allocations = make([]dto.TripAllocationResponse, 0, len(trip.Allocations))
	for _, a := range trip.Allocations {
		resp.Allocations = append(resp.Allocations, mapAllocation(&a, trip.DaysRemaining()))
	}

	resp.Itinerary = make([]dto.TripItineraryResponse, 0, len(trip.Itinerary))
	for _, i := range trip.Itinerary {
		resp.Itinerary = append(resp.Itinerary, mapItinerary(&i))
	}

	return resp
}

func mapMember(m *entity.TripMember) dto.TripMemberResponse {
	out := dto.TripMemberResponse{
		ID:          m.ID,
		UserID:      m.UserID,
		DisplayName: m.DisplayName,
		AvatarURL:   m.AvatarURL,
		Role:        string(m.Role),
		IsGhost:     m.IsGhost,
		JoinedAt:    m.JoinedAt,
	}
	if m.Email != nil {
		out.Email = *m.Email
	}
	return out
}

func mapAllocation(a *entity.TripBudgetAllocation, remainingDays int) dto.TripAllocationResponse {
	out := dto.TripAllocationResponse{
		ID:              a.ID,
		EstimatedAmount: a.EstimatedAmount,
		SpentAmount:     a.SpentAmount,
		RemainingAmount: a.RemainingAmount(),
		ProgressPercent: a.ProgressPercentage(),
		DailySuggested:  a.DailySuggested(remainingDays),
		IsOverBudget:    a.IsOverBudget(),
		Currency:        a.Currency,
		Notes:           a.Notes,
	}
	if a.Category != nil {
		out.Category = dto.CategorySummaryResponse{
			ID:             a.Category.ID,
			Name:           a.Category.Name,
			Description:    a.Category.Description,
			Icon:           a.Category.Icon,
			Color:          a.Category.Color,
			DisplayName:    a.Category.GetDisplayName(),
			IsActive:       a.Category.IsActive,
			IsDefault:      a.Category.IsDefault,
			IsUserCategory: a.Category.IsUserCategory(),
			SortOrder:      a.Category.SortOrder,
			CanBeDeleted:   a.Category.CanBeDeleted(),
		}
	}
	return out
}

func mapItinerary(i *entity.TripItineraryItem) dto.TripItineraryResponse {
	out := dto.TripItineraryResponse{
		ID:            i.ID,
		Day:           i.Day,
		Time:          i.Time,
		Type:          string(i.Type),
		Title:         i.Title,
		Description:   i.Description,
		Location:      i.Location,
		EstimatedCost: i.EstimatedCost,
		Currency:      i.Currency,
		ExpenseID:     i.ExpenseID,
	}
	if i.Expense != nil {
		out.ActualAmount = i.Expense.Amount
		out.Variance = i.Variance(i.Expense.Amount)
	}
	return out
}

// generateInvitationToken genera un token aleatorio criptográficamente seguro
func generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate invitation token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ensureValidStatus valida el string recibido contra los TripStatus válidos
func ensureValidStatus(status string) (entity.TripStatus, error) {
	switch entity.TripStatus(status) {
	case entity.TripStatusPlanning, entity.TripStatusActive,
		entity.TripStatusCompleted, entity.TripStatusCancelled:
		return entity.TripStatus(status), nil
	default:
		return "", errors.New("invalid trip status")
	}
}
