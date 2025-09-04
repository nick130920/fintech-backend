package usecase

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// ExpenseUseCase contiene la lógica de negocio para gastos
type ExpenseUseCase struct {
	expenseRepo  repo.ExpenseRepo
	budgetRepo   repo.BudgetRepo
	categoryRepo repo.CategoryRepo
	userRepo     repo.UserRepo
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
	}
}

// CreateExpense crea un nuevo gasto
func (uc *ExpenseUseCase) CreateExpense(userID uint, req *dto.CreateExpenseRequest) (*dto.ExpenseSummaryResponse, error) {
	log.Printf("🔍 ExpenseUseCase.CreateExpense iniciado - UserID: %d", userID)

	// Verificar que el usuario existe
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("❌ Usuario no encontrado: %d | Error: %v", userID, err)
		return nil, errors.New("user not found")
	}

	log.Printf("✅ Usuario encontrado: %s %s (ID: %d)", user.FirstName, user.LastName, user.ID)

	if !user.IsAccountActive() {
		log.Printf("❌ Cuenta de usuario inactiva: %d", userID)
		return nil, errors.New("user account is not active")
	}

	// Verificar que la categoría existe
	log.Printf("🔍 Buscando categoría: %d", req.CategoryID)
	category, err := uc.categoryRepo.GetByID(req.CategoryID)
	if err != nil {
		log.Printf("❌ Categoría no encontrada: %d | Error: %v", req.CategoryID, err)
		return nil, errors.New("category not found")
	}

	log.Printf("✅ Categoría encontrada: %s (ID: %d)", category.Name, category.ID)

	// Verificar que la categoría pertenece al usuario o es del sistema
	if !category.IsSystemCategory() && category.UserID != nil && *category.UserID != userID {
		return nil, errors.New("category not found")
	}

	// Parsear fecha con múltiples formatos soportados
	log.Printf("🔍 Parseando fecha: %s", req.Date)
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
		log.Printf("❌ No se pudo parsear fecha con ningún formato: %s | Error: %v", req.Date, err)
		return nil, fmt.Errorf("invalid date format: %s", req.Date)
	}

	log.Printf("✅ Fecha parseada: %v", expenseDate)

	// Intentar obtener el presupuesto del mes del gasto, si no existe usar el actual
	log.Printf("🔍 Buscando presupuesto para: %d/%d, Usuario: %d", expenseDate.Year(), int(expenseDate.Month()), userID)
	budget, err := uc.budgetRepo.GetByUserAndMonth(userID, expenseDate.Year(), int(expenseDate.Month()))

	// Si no existe presupuesto para ese mes, intentar usar el presupuesto actual
	if err != nil {
		log.Printf("⚠️ Presupuesto no encontrado para %d/%d, intentando usar presupuesto actual", expenseDate.Year(), int(expenseDate.Month()))
		currentBudget, currentErr := uc.budgetRepo.GetCurrentBudget(userID)
		if currentErr != nil {
			log.Printf("❌ Tampoco se encontró presupuesto actual | Error: %v", currentErr)
			return nil, fmt.Errorf("no budget found for date %d/%d and no current budget exists: %v", expenseDate.Year(), int(expenseDate.Month()), err)
		}
		budget = currentBudget
		log.Printf("✅ Usando presupuesto actual: ID=%d, Período=%d/%d", budget.ID, budget.Year, budget.Month)
	} else {
		log.Printf("✅ Presupuesto encontrado: ID=%d, Total=%.2f", budget.ID, budget.TotalAmount)
	}

	// Obtener la asignación de la categoría en el presupuesto
	log.Printf("🔍 Buscando asignación: Budget=%d, Category=%d", budget.ID, category.ID)
	allocation, err := uc.budgetRepo.GetAllocationByBudgetAndCategory(budget.ID, category.ID)
	if err != nil {
		log.Printf("❌ Asignación no encontrada: Budget=%d, Category=%d | Error: %v", budget.ID, category.ID, err)
		return nil, fmt.Errorf("category not allocated in budget: %v", err)
	}

	log.Printf("✅ Asignación encontrada: ID=%d, Asignado=%.2f, Gastado=%.2f", allocation.ID, allocation.AllocatedAmount, allocation.SpentAmount)

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

	log.Printf("🔍 Creando gasto en base de datos: %+v", expense)
	if err := uc.expenseRepo.Create(expense); err != nil {
		log.Printf("❌ Error al crear gasto en DB: %v", err)
		return nil, fmt.Errorf("error creating expense: %v", err)
	}

	log.Printf("✅ Gasto creado en DB con ID: %d", expense.ID)

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
