package usecase

import (
	"errors"
	"math"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"github.com/nick130920/fintech-backend/pkg/exchange"
	"github.com/nick130920/fintech-backend/pkg/logger"
	"github.com/rs/zerolog"
)

// TripExpenseUseCase administra gastos vinculados a un viaje y sus splits
type TripExpenseUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	expenseRepo    repo.ExpenseRepo
	splitRepo      repo.ExpenseSplitRepo
	allocationRepo repo.TripBudgetAllocationRepo
	categoryRepo   repo.CategoryRepo
	fxProvider     exchange.Provider
	logger         zerolog.Logger
}

// NewTripExpenseUseCase construye el use case de gastos del viaje
func NewTripExpenseUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	expenseRepo repo.ExpenseRepo,
	splitRepo repo.ExpenseSplitRepo,
	allocationRepo repo.TripBudgetAllocationRepo,
	categoryRepo repo.CategoryRepo,
	fxProvider exchange.Provider,
) *TripExpenseUseCase {
	return &TripExpenseUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		expenseRepo:    expenseRepo,
		splitRepo:      splitRepo,
		allocationRepo: allocationRepo,
		categoryRepo:   categoryRepo,
		fxProvider:     fxProvider,
		logger:         logger.Get().With().Str("usecase", "TripExpense").Logger(),
	}
}

// ListExpenses retorna los gastos del viaje
func (uc *TripExpenseUseCase) ListExpenses(userID, tripID uint) ([]*dto.TripExpenseResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	expenses, err := uc.expenseRepo.GetByTripID(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	out := make([]*dto.TripExpenseResponse, 0, len(expenses))
	for _, exp := range expenses {
		out = append(out, mapTripExpense(exp, trip))
	}
	return out, nil
}

// CreateExpense crea un gasto vinculado al viaje con sus splits
func (uc *TripExpenseUseCase) CreateExpense(userID, tripID uint, req *dto.CreateTripExpenseRequest) (*dto.TripExpenseResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}
	if !uc.canRegisterExpenses(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	category, err := uc.categoryRepo.GetByID(req.CategoryID)
	if err != nil || category == nil {
		return nil, apperrors.ErrCategoryNotFound
	}

	payer, err := uc.memberRepo.GetByTripAndID(tripID, req.PaidByMemberID)
	if err != nil || payer == nil {
		return nil, ErrTripMemberNotFound
	}

	currency := req.Currency
	if currency == "" {
		currency = trip.PrimaryCurrency
	}

	exchangeRate := req.ExchangeRate
	if exchangeRate <= 0 {
		exchangeRate = uc.resolveFxRate(currency, trip.PrimaryCurrency)
	}

	tripIDValue := tripID
	paidByID := req.PaidByMemberID
	expense := &entity.Expense{
		UserID:         userID,
		CategoryID:     category.ID,
		Amount:         req.Amount,
		Description:    req.Description,
		Date:           req.Date,
		Source:         entity.ExpenseSourceManual,
		Status:         entity.ExpenseStatusConfirmed,
		Location:       req.Location,
		Merchant:       req.Merchant,
		Notes:          req.Notes,
		ReceiptURL:     req.ReceiptURL,
		Currency:       currency,
		ExchangeRate:   exchangeRate,
		TripID:         &tripIDValue,
		PaidByMemberID: &paidByID,
	}

	if err := uc.expenseRepo.Create(expense); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	splits, err := uc.buildSplits(expense.ID, req.Amount, req.Splits)
	if err != nil {
		return nil, err
	}
	if err := uc.splitRepo.BatchCreate(splits); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	if err := uc.refreshTripTotals(trip); err != nil {
		return nil, err
	}

	loaded, err := uc.expenseRepo.GetByID(expense.ID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	loaded.Splits = entityFromSplits(splits)
	loaded.PaidBy = payer

	return mapTripExpense(loaded, trip), nil
}

// UpdateExpense actualiza un gasto y opcionalmente recalcula sus splits
func (uc *TripExpenseUseCase) UpdateExpense(userID, tripID, expenseID uint, req *dto.UpdateTripExpenseRequest) (*dto.TripExpenseResponse, error) {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	expense, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil || expense == nil || expense.TripID == nil || *expense.TripID != tripID {
		return nil, ErrTripExpenseNotFound
	}

	if req.CategoryID != nil {
		expense.CategoryID = *req.CategoryID
	}
	if req.Amount != nil {
		expense.Amount = *req.Amount
	}
	if req.Description != nil {
		expense.Description = *req.Description
	}
	if req.Date != nil {
		expense.Date = *req.Date
	}
	if req.Currency != nil {
		expense.Currency = *req.Currency
	}
	if req.ExchangeRate != nil {
		expense.ExchangeRate = *req.ExchangeRate
	}
	if req.Location != nil {
		expense.Location = *req.Location
	}
	if req.Merchant != nil {
		expense.Merchant = *req.Merchant
	}
	if req.Notes != nil {
		expense.Notes = *req.Notes
	}
	if req.ReceiptURL != nil {
		expense.ReceiptURL = *req.ReceiptURL
	}
	if req.PaidByMemberID != nil {
		paidBy, errMember := uc.memberRepo.GetByTripAndID(tripID, *req.PaidByMemberID)
		if errMember != nil || paidBy == nil {
			return nil, ErrTripMemberNotFound
		}
		expense.PaidByMemberID = req.PaidByMemberID
	}

	if err := uc.expenseRepo.Update(expense); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	if len(req.Splits) > 0 {
		if err := uc.splitRepo.DeleteByExpense(expenseID); err != nil {
			return nil, apperrors.ErrInternal.WithInternal(err)
		}
		newSplits, errBuild := uc.buildSplits(expenseID, expense.Amount, req.Splits)
		if errBuild != nil {
			return nil, errBuild
		}
		if err := uc.splitRepo.BatchCreate(newSplits); err != nil {
			return nil, apperrors.ErrInternal.WithInternal(err)
		}
	}

	if err := uc.refreshTripTotals(trip); err != nil {
		return nil, err
	}

	loaded, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	splits, _ := uc.splitRepo.GetByExpense(expenseID)
	loaded.Splits = entityFromSplitsPtr(splits)

	return mapTripExpense(loaded, trip), nil
}

// DeleteExpense elimina un gasto del viaje (y sus splits por cascada)
func (uc *TripExpenseUseCase) DeleteExpense(userID, tripID, expenseID uint) error {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return ErrTripPermissionDenied
	}

	expense, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil || expense == nil || expense.TripID == nil || *expense.TripID != tripID {
		return ErrTripExpenseNotFound
	}

	if err := uc.splitRepo.DeleteByExpense(expenseID); err != nil {
		return apperrors.ErrInternal.WithInternal(err)
	}
	if err := uc.expenseRepo.Delete(expenseID); err != nil {
		return apperrors.ErrInternal.WithInternal(err)
	}
	return uc.refreshTripTotals(trip)
}

// ----- Helpers internos -----

// buildSplits crea entidades ExpenseSplit a partir de la entrada del cliente
func (uc *TripExpenseUseCase) buildSplits(expenseID uint, total float64, inputs []dto.ExpenseSplitInput) ([]*entity.ExpenseSplit, error) {
	if len(inputs) == 0 {
		return nil, apperrors.ErrInvalidRequest.WithDetails("splits requeridos")
	}
	splits := make([]*entity.ExpenseSplit, 0, len(inputs))
	for _, input := range inputs {
		splits = append(splits, &entity.ExpenseSplit{
			ExpenseID:  expenseID,
			MemberID:   input.MemberID,
			ShareType:  entity.ExpenseSplitShareType(input.ShareType),
			ShareValue: input.ShareValue,
		})
	}
	if err := entity.RecalculateShares(splits, total); err != nil {
		return nil, apperrors.ErrInvalidRequest.WithDetails(err.Error())
	}

	// Validación final: la suma debe coincidir con el total
	sum := 0.0
	for _, s := range splits {
		sum += s.ShareAmount
	}
	if math.Abs(sum-total) > 0.05 {
		return nil, ErrTripSplitsMismatch
	}
	return splits, nil
}

// refreshTripTotals recalcula totales del trip y allocations sumando gastos
func (uc *TripExpenseUseCase) refreshTripTotals(trip *entity.Trip) error {
	expenses, err := uc.expenseRepo.GetByTripID(trip.ID)
	if err != nil {
		return apperrors.ErrInternal.WithInternal(err)
	}

	totalSpent := 0.0
	byCategory := map[uint]float64{}
	for _, expense := range expenses {
		amount := expense.Amount
		if expense.Currency != trip.PrimaryCurrency && expense.ExchangeRate > 0 {
			amount = expense.Amount * expense.ExchangeRate
		}
		totalSpent += amount
		byCategory[expense.CategoryID] += amount
	}

	allocations, err := uc.allocationRepo.GetByTrip(trip.ID)
	if err != nil {
		return apperrors.ErrInternal.WithInternal(err)
	}
	for _, allocation := range allocations {
		if err := uc.allocationRepo.UpdateSpent(allocation.ID, byCategory[allocation.CategoryID]); err != nil {
			return apperrors.ErrInternal.WithInternal(err)
		}
	}

	return uc.tripRepo.UpdateTotals(trip.ID, trip.EstimatedTotal, totalSpent)
}

// resolveFxRate consulta el provider y devuelve 1 ante errores
func (uc *TripExpenseUseCase) resolveFxRate(from, to string) float64 {
	if from == to || uc.fxProvider == nil {
		return 1
	}
	rate, err := uc.fxProvider.GetRate(from, to)
	if err != nil || rate == nil {
		return 1
	}
	return rate.Rate
}

func (uc *TripExpenseUseCase) hasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	return err == nil && member != nil
}

func (uc *TripExpenseUseCase) canRegisterExpenses(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	if err != nil || member == nil {
		return false
	}
	return member.CanRegisterExpenses()
}

// mapTripExpense convierte el gasto + splits en una respuesta para el cliente
func mapTripExpense(expense *entity.Expense, trip *entity.Trip) *dto.TripExpenseResponse {
	if expense == nil {
		return nil
	}
	resp := &dto.TripExpenseResponse{
		ID:           expense.ID,
		CategoryID:   expense.CategoryID,
		Amount:       expense.Amount,
		Currency:     expense.Currency,
		ExchangeRate: expense.ExchangeRate,
		Description:  expense.Description,
		Date:         expense.Date,
		Location:     expense.Location,
		Merchant:     expense.Merchant,
		Notes:        expense.Notes,
		ReceiptURL:   expense.ReceiptURL,
		CreatedAt:    expense.CreatedAt,
		UpdatedAt:    expense.UpdatedAt,
	}
	if expense.TripID != nil {
		resp.TripID = *expense.TripID
	}

	// Conversión a moneda primaria del viaje
	resp.AmountPrimary = expense.Amount
	if trip != nil && expense.Currency != trip.PrimaryCurrency && expense.ExchangeRate > 0 {
		resp.AmountPrimary = expense.Amount * expense.ExchangeRate
	}

	// Categoría embebida
	resp.Category = dto.CategorySummaryResponse{
		ID:             expense.Category.ID,
		Name:           expense.Category.Name,
		Description:    expense.Category.Description,
		Icon:           expense.Category.Icon,
		Color:          expense.Category.Color,
		DisplayName:    expense.Category.GetDisplayName(),
		IsActive:       expense.Category.IsActive,
		IsDefault:      expense.Category.IsDefault,
		IsUserCategory: expense.Category.IsUserCategory(),
		SortOrder:      expense.Category.SortOrder,
		CanBeDeleted:   expense.Category.CanBeDeleted(),
	}

	if expense.PaidByMemberID != nil {
		resp.PaidByMemberID = expense.PaidByMemberID
	}
	if expense.PaidBy != nil {
		resp.PaidByName = expense.PaidBy.DisplayName
	}

	resp.Splits = make([]dto.TripExpenseSplit, 0, len(expense.Splits))
	for _, split := range expense.Splits {
		entry := dto.TripExpenseSplit{
			ID:          split.ID,
			MemberID:    split.MemberID,
			ShareType:   string(split.ShareType),
			ShareValue:  split.ShareValue,
			ShareAmount: split.ShareAmount,
			IsPaid:      split.IsPaid,
		}
		if split.Member != nil {
			entry.MemberName = split.Member.DisplayName
		}
		resp.Splits = append(resp.Splits, entry)
	}

	return resp
}

func entityFromSplits(splits []*entity.ExpenseSplit) []entity.ExpenseSplit {
	out := make([]entity.ExpenseSplit, 0, len(splits))
	for _, s := range splits {
		out = append(out, *s)
	}
	return out
}

func entityFromSplitsPtr(splits []*entity.ExpenseSplit) []entity.ExpenseSplit {
	out := make([]entity.ExpenseSplit, 0, len(splits))
	for _, s := range splits {
		if s == nil {
			continue
		}
		out = append(out, *s)
	}
	return out
}

// SplitTypeFromString convierte un string a un ExpenseSplitShareType validado
func SplitTypeFromString(s string) (entity.ExpenseSplitShareType, error) {
	switch entity.ExpenseSplitShareType(s) {
	case entity.ExpenseSplitShareTypeEqual,
		entity.ExpenseSplitShareTypePercentage,
		entity.ExpenseSplitShareTypeExact,
		entity.ExpenseSplitShareTypeShares:
		return entity.ExpenseSplitShareType(s), nil
	}
	return "", errors.New("invalid share type")
}
