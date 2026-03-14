package usecase

import (
	"errors"
	"fmt"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/logger"
	"github.com/rs/zerolog"
)

// ExpenseUseCase contiene la lógica de negocio para gastos
type ExpenseUseCase struct {
	expenseRepo  repo.ExpenseRepo
	budgetRepo   repo.BudgetRepo
	categoryRepo repo.CategoryRepo
	userRepo     repo.UserRepo
	logger       zerolog.Logger
}

// NewExpenseUseCase crea una nueva instancia de ExpenseUseCase
func NewExpenseUseCase(
	expenseRepo repo.ExpenseRepo,
	budgetRepo repo.BudgetRepo,
	categoryRepo repo.CategoryRepo,
	userRepo repo.UserRepo,
) *ExpenseUseCase {
	return &ExpenseUseCase{
		expenseRepo:  expenseRepo,
		budgetRepo:   budgetRepo,
		categoryRepo: categoryRepo,
		userRepo:     userRepo,
		logger:       logger.Get().With().Str("usecase", "Expense").Logger(),
	}
}

// CreateExpense crea un nuevo gasto
func (uc *ExpenseUseCase) CreateExpense(userID uint, req *dto.CreateExpenseRequest) (*dto.ExpenseSummaryResponse, error) {
	log := uc.logger.With().Uint("user_id", userID).Logger()
	log.Debug().Msg("CreateExpense started")

	// Verificar que el usuario existe
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		log.Warn().Err(err).Msg("User not found")
		return nil, errors.New("user not found")
	}

	log.Debug().Msgf("User found: %s %s", user.FirstName, user.LastName)

	if !user.IsAccountActive() {
		log.Warn().Msg("User account is not active")
		return nil, errors.New("user account is not active")
	}

	// Verificar que la categoría existe
	log.Debug().Uint("category_id", req.CategoryID).Msg("Searching for category")
	category, err := uc.categoryRepo.GetByID(req.CategoryID)
	if err != nil {
		log.Warn().Err(err).Uint("category_id", req.CategoryID).Msg("Category not found")
		return nil, errors.New("category not found")
	}

	log.Debug().Str("category_name", category.Name).Msg("Category found")

	// Verificar que la categoría pertenece al usuario o es del sistema
	if !category.IsSystemCategory() && category.UserID != nil && *category.UserID != userID {
		log.Warn().Uint("category_id", category.ID).Msg("Category does not belong to user")
		return nil, errors.New("category not found")
	}

	// Parsear fecha con múltiples formatos soportados
	log.Debug().Str("date_string", req.Date).Msg("Parsing date")
	var expenseDate time.Time

	// Intentar diferentes formatos de fecha
	dateFormats := []string{
		time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02T15:04:05.999999", // Formato con microsegundos (Flutter)
		"2006-01-02T15:04:05",        // ISO sin timezone
		"2006-01-02",                 // Solo fecha
	}

	for _, format := range dateFormats {
		if expenseDate, err = time.Parse(format, req.Date); err == nil {
			break
		}
	}

	if err != nil {
		log.Warn().Err(err).Str("date_string", req.Date).Msg("Could not parse date with any format")
		return nil, fmt.Errorf("invalid date format: %s", req.Date)
	}

	log.Debug().Time("expense_date", expenseDate).Msg("Date parsed successfully")

	// Intentar obtener el presupuesto del mes del gasto, si no existe usar el actual
	log.Debug().Int("year", expenseDate.Year()).Int("month", int(expenseDate.Month())).Msg("Searching for budget")
	budget, err := uc.budgetRepo.GetByUserAndMonth(userID, expenseDate.Year(), int(expenseDate.Month()))

	// Si no existe presupuesto para ese mes, intentar usar el presupuesto actual
	if err != nil {
		log.Warn().Err(err).Int("year", expenseDate.Year()).Int("month", int(expenseDate.Month())).Msg("Budget for expense month not found, trying current budget")
		currentBudget, currentErr := uc.budgetRepo.GetCurrentBudget(userID)
		if currentErr != nil {
			log.Error().Err(currentErr).Msg("Current budget also not found")
			return nil, fmt.Errorf("no budget found for date %d/%d and no current budget exists: %v", expenseDate.Year(), int(expenseDate.Month()), err)
		}
		budget = currentBudget
		log.Info().Uint("budget_id", budget.ID).Msg("Using current budget")
	} else {
		log.Debug().Uint("budget_id", budget.ID).Float64("total_amount", budget.TotalAmount).Msg("Budget found")
	}

	// Obtener la asignación de la categoría en el presupuesto
	log.Debug().Uint("budget_id", budget.ID).Uint("category_id", category.ID).Msg("Searching for allocation")
	allocation, err := uc.budgetRepo.GetAllocationByBudgetAndCategory(budget.ID, category.ID)
	if err != nil {
		log.Warn().Err(err).Uint("budget_id", budget.ID).Uint("category_id", category.ID).Msg("Allocation not found")
		return nil, fmt.Errorf("category not allocated in budget: %v", err)
	}

	log.Debug().
		Uint("allocation_id", allocation.ID).
		Float64("allocated", allocation.AllocatedAmount).
		Float64("spent", allocation.SpentAmount).
		Msg("Allocation found")

	// Crear el gasto
	expense := &entity.Expense{
		UserID:       userID,
		BudgetID:     budget.ID,
		CategoryID:   category.ID,
		AllocationID: allocation.ID,
		Amount:       req.Amount,
		Description:  req.Description,
		Date:         expenseDate,
		Source:       req.Source,
		Status:       entity.ExpenseStatusConfirmed,
		Location:     req.Location,
		Merchant:     req.Merchant,
		Notes:        req.Notes,
		Currency:     user.Currency, // Usar divisa del usuario
		ReceiptURL:   req.ReceiptURL,
	}

	// Si no se especifica source, usar manual
	if expense.Source == "" {
		expense.Source = entity.ExpenseSourceManual
	}

	log.Debug().Interface("expense", expense).Msg("Creating expense in database")
	if err := uc.expenseRepo.Create(expense); err != nil {
		log.Error().Err(err).Msg("Error creating expense in DB")
		return nil, fmt.Errorf("error creating expense: %v", err)
	}

	log.Info().Uint("expense_id", expense.ID).Msg("Expense created in DB successfully")

	// Actualizar montos gastados
	if err := uc.budgetRepo.UpdateAllocationSpentAmount(allocation.ID); err != nil {
		return nil, err
	}

	if err := uc.budgetRepo.UpdateBudgetSpentAmount(budget.ID); err != nil {
		return nil, err
	}

	// Recargar con relaciones
	expenseWithRelations, err := uc.expenseRepo.GetByID(expense.ID)
	if err != nil {
		return nil, err
	}

	return uc.mapExpenseToSummaryResponse(expenseWithRelations), nil
}

// GetExpenses obtiene gastos con filtros
func (uc *ExpenseUseCase) GetExpenses(userID uint, categoryID *uint, startDate, endDate *time.Time, limit, offset int) ([]*dto.ExpenseSummaryResponse, error) {
	var expenses []*entity.Expense
	var err error

	if categoryID != nil {
		expenses, err = uc.expenseRepo.GetByCategoryAndDateRange(userID, *categoryID, startDate, endDate)
	} else {
		expenses, err = uc.expenseRepo.GetByUserAndDateRange(userID, startDate, endDate)
	}

	if err != nil {
		return nil, err
	}

	// Aplicar paginación manual (simple)
	if offset >= len(expenses) {
		return []*dto.ExpenseSummaryResponse{}, nil
	}

	end := offset + limit
	if end > len(expenses) {
		end = len(expenses)
	}

	paginatedExpenses := expenses[offset:end]

	// Mapear a DTOs
	response := make([]*dto.ExpenseSummaryResponse, len(paginatedExpenses))
	for i, expense := range paginatedExpenses {
		response[i] = uc.mapExpenseToSummaryResponse(expense)
	}

	return response, nil
}

// GetExpensesByCategory obtiene resumen de gastos por categoría
func (uc *ExpenseUseCase) GetExpensesByCategory(userID uint) (map[string]interface{}, error) {
	// Obtener gastos del mes actual
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	expenses, err := uc.expenseRepo.GetByUserAndDateRange(userID, &startOfMonth, &endOfMonth)
	if err != nil {
		return nil, err
	}

	// Agrupar por categoría
	categoryTotals := make(map[uint]float64)
	categoryNames := make(map[uint]string)
	categoryIcons := make(map[uint]string)

	for _, expense := range expenses {
		if expense.IsConfirmed() {
			categoryTotals[expense.CategoryID] += expense.Amount
			categoryNames[expense.CategoryID] = expense.Category.Name
			categoryIcons[expense.CategoryID] = expense.Category.Icon
		}
	}

	// Construir respuesta
	categories := make([]map[string]interface{}, 0)
	for categoryID, total := range categoryTotals {
		categories = append(categories, map[string]interface{}{
			"category_id":   categoryID,
			"category_name": categoryNames[categoryID],
			"category_icon": categoryIcons[categoryID],
			"total_amount":  total,
		})
	}

	return map[string]interface{}{
		"categories":  categories,
		"month":       int(now.Month()),
		"year":        now.Year(),
		"total_spent": uc.calculateTotalAmount(expenses),
	}, nil
}

// GetRecentExpenses obtiene gastos recientes
func (uc *ExpenseUseCase) GetRecentExpenses(userID uint, limit int) ([]*dto.ExpenseSummaryResponse, error) {
	expenses, err := uc.expenseRepo.GetRecentExpenses(userID, limit)
	if err != nil {
		return nil, err
	}

	// Mapear a DTOs
	response := make([]*dto.ExpenseSummaryResponse, len(expenses))
	for i, expense := range expenses {
		response[i] = uc.mapExpenseToSummaryResponse(expense)
	}

	return response, nil
}

// UpdateExpense actualiza un gasto existente
func (uc *ExpenseUseCase) UpdateExpense(userID, expenseID uint, req *dto.UpdateExpenseRequest) (*dto.ExpenseSummaryResponse, error) {
	expense, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil {
		return nil, errors.New("expense not found")
	}

	// Verificar que pertenece al usuario
	if expense.UserID != userID {
		return nil, errors.New("expense not found")
	}

	// Verificar que puede ser modificado
	if !expense.CanBeModified() {
		return nil, errors.New("expense cannot be modified")
	}

	// Actualizar campos
	if req.CategoryID != nil {
		// Verificar que la nueva categoría existe
		category, err := uc.categoryRepo.GetByID(*req.CategoryID)
		if err != nil {
			return nil, errors.New("category not found")
		}

		// Verificar que la categoría pertenece al usuario o es del sistema
		if !category.IsSystemCategory() && category.UserID != nil && *category.UserID != userID {
			return nil, errors.New("category not found")
		}

		expense.CategoryID = *req.CategoryID
	}

	if req.Amount != nil {
		expense.Amount = *req.Amount
	}

	if req.Description != "" {
		expense.Description = req.Description
	}

	if req.Date != "" {
		expenseDate, err := time.Parse(time.RFC3339, req.Date)
		if err != nil {
			expenseDate, err = time.Parse("2006-01-02", req.Date)
			if err != nil {
				return nil, errors.New("invalid date format")
			}
		}
		expense.Date = expenseDate
	}

	if req.Location != "" {
		expense.Location = req.Location
	}

	if req.Merchant != "" {
		expense.Merchant = req.Merchant
	}

	if req.Notes != "" {
		expense.Notes = req.Notes
	}

	if req.ReceiptURL != "" {
		expense.ReceiptURL = req.ReceiptURL
	}

	if len(req.Tags) > 0 {
		expense.SetTags(req.Tags)
	}

	if err := uc.expenseRepo.Update(expense); err != nil {
		return nil, err
	}

	// Actualizar montos gastados
	if err := uc.budgetRepo.UpdateAllocationSpentAmount(expense.AllocationID); err != nil {
		return nil, err
	}

	if err := uc.budgetRepo.UpdateBudgetSpentAmount(expense.BudgetID); err != nil {
		return nil, err
	}

	// Recargar con relaciones
	updatedExpense, err := uc.expenseRepo.GetByID(expense.ID)
	if err != nil {
		return nil, err
	}

	return uc.mapExpenseToSummaryResponse(updatedExpense), nil
}

// DeleteExpense elimina un gasto
func (uc *ExpenseUseCase) DeleteExpense(userID, expenseID uint) error {
	expense, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil {
		return errors.New("expense not found")
	}

	// Verificar que pertenece al usuario
	if expense.UserID != userID {
		return errors.New("expense not found")
	}

	// Verificar que puede ser eliminado
	if !expense.CanBeCancelled() {
		return errors.New("expense cannot be deleted")
	}

	if err := uc.expenseRepo.Delete(expenseID); err != nil {
		return err
	}

	// Actualizar montos gastados
	if err := uc.budgetRepo.UpdateAllocationSpentAmount(expense.AllocationID); err != nil {
		return err
	}

	if err := uc.budgetRepo.UpdateBudgetSpentAmount(expense.BudgetID); err != nil {
		return err
	}

	return nil
}

// Helper methods

func (uc *ExpenseUseCase) mapExpenseToSummaryResponse(expense *entity.Expense) *dto.ExpenseSummaryResponse {
	return &dto.ExpenseSummaryResponse{
		ID:              expense.ID,
		Amount:          expense.Amount,
		FormattedAmount: expense.GetFormattedAmount(),
		Description:     expense.Description,
		Date:            expense.Date.Format(time.RFC3339),
		TimeAgo:         expense.GetTimeAgo(),
		Category:        uc.mapCategoryToSummaryResponse(&expense.Category),
		Source:          expense.Source,
		Status:          expense.Status,
		Location:        expense.Location,
		Merchant:        expense.Merchant,
		Tags:            expense.GetTags(),
		Notes:           expense.Notes,
		Currency:        expense.Currency,
		CanBeModified:   expense.CanBeModified(),
		CanBeCancelled:  expense.CanBeCancelled(),
		TriggeredAlert:  expense.TriggeredAlert,
		CreatedAt:       expense.CreatedAt.Format(time.RFC3339),
	}
}

func (uc *ExpenseUseCase) mapCategoryToSummaryResponse(category *entity.Category) dto.CategorySummaryResponse {
	return dto.CategorySummaryResponse{
		ID:             category.ID,
		Name:           category.Name,
		Description:    category.Description,
		Icon:           category.Icon,
		Color:          category.Color,
		DisplayName:    category.GetDisplayName(),
		IsActive:       category.IsActive,
		IsDefault:      category.IsDefault,
		IsUserCategory: category.IsUserCategory(),
		SortOrder:      category.SortOrder,
		CanBeDeleted:   category.CanBeDeleted(),
	}
}

func (uc *ExpenseUseCase) calculateTotalAmount(expenses []*entity.Expense) float64 {
	total := float64(0)
	for _, expense := range expenses {
		if expense.IsConfirmed() {
			total += expense.Amount
		}
	}
	return total
}

// GetAutomaticExpenses obtiene los gastos creados automáticamente por IA/notificaciones
func (uc *ExpenseUseCase) GetAutomaticExpenses(userID uint, limit int) ([]*dto.ExpenseSummaryResponse, error) {
	expenses, err := uc.expenseRepo.GetPendingExpenses(userID)
	if err != nil {
		return nil, err
	}

	// Filtrar solo los gastos automáticos (source = notification o sms)
	var automaticExpenses []*entity.Expense
	for _, expense := range expenses {
		if expense.Source == entity.ExpenseSourceNotification ||
			expense.Source == entity.ExpenseSourceSMS ||
			expense.Source == entity.ExpenseSourceBankAPI {
			automaticExpenses = append(automaticExpenses, expense)
		}
	}

	// Limitar resultados
	if limit > 0 && len(automaticExpenses) > limit {
		automaticExpenses = automaticExpenses[:limit]
	}

	// Convertir a DTOs
	responses := make([]*dto.ExpenseSummaryResponse, len(automaticExpenses))
	for i, expense := range automaticExpenses {
		responses[i] = uc.mapExpenseToSummaryResponse(expense)
	}

	return responses, nil
}

// ConfirmExpense confirma un gasto pendiente
func (uc *ExpenseUseCase) ConfirmExpense(userID, expenseID uint) (*dto.ExpenseSummaryResponse, error) {
	// Obtener el gasto
	expense, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, errors.New("expense not found")
	}
	if expense.UserID != userID {
		return nil, errors.New("unauthorized")
	}
	if expense.Status != entity.ExpenseStatusPending {
		return nil, errors.New("expense is not pending")
	}

	// Confirmar el gasto
	if err := uc.expenseRepo.ConfirmExpense(expenseID); err != nil {
		return nil, err
	}

	// Actualizar el gasto en memoria y obtener la versión actualizada
	expense.Status = entity.ExpenseStatusConfirmed

	// Actualizar el monto gastado en la asignación
	if err := uc.budgetRepo.UpdateAllocationSpentAmount(expense.AllocationID); err != nil {
		// Log pero no fallar
	}

	return uc.mapExpenseToSummaryResponse(expense), nil
}

// RejectExpense rechaza/cancela un gasto pendiente
func (uc *ExpenseUseCase) RejectExpense(userID, expenseID uint) error {
	// Obtener el gasto
	expense, err := uc.expenseRepo.GetByID(expenseID)
	if err != nil {
		return err
	}
	if expense == nil {
		return errors.New("expense not found")
	}
	if expense.UserID != userID {
		return errors.New("unauthorized")
	}
	if expense.Status != entity.ExpenseStatusPending {
		return errors.New("expense is not pending")
	}

	// Cancelar el gasto
	return uc.expenseRepo.CancelExpense(expenseID)
}
