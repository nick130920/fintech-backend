package usecase

import (
	"fmt"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// BudgetUseCase contiene la lógica de negocio para presupuestos
type BudgetUseCase struct {
	budgetRepo   repo.BudgetRepo
	categoryRepo repo.CategoryRepo
	expenseRepo  repo.ExpenseRepo
	userRepo     repo.UserRepo
}

// NewBudgetUseCase crea una nueva instancia de BudgetUseCase
func NewBudgetUseCase(
	budgetRepo repo.BudgetRepo,
	categoryRepo repo.CategoryRepo,
	expenseRepo repo.ExpenseRepo,
	userRepo repo.UserRepo,
) *BudgetUseCase {
	return &BudgetUseCase{
		budgetRepo:   budgetRepo,
		categoryRepo: categoryRepo,
		expenseRepo:  expenseRepo,
		userRepo:     userRepo,
	}
}

// CreateBudget crea un nuevo presupuesto mensual
func (uc *BudgetUseCase) CreateBudget(userID uint, req *dto.CreateBudgetRequest) (*dto.BudgetSummaryResponse, error) {
	// Verificar que el usuario existe
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	if !user.IsAccountActive() {
		return nil, apperrors.ErrAccountInactive
	}

	// Verificar que no existe ya un presupuesto para ese mes
	existingBudget, _ := uc.budgetRepo.GetByUserAndMonth(userID, req.Year, req.Month)
	if existingBudget != nil {
		return nil, apperrors.ErrBudgetExists
	}

	// Validar que la suma de asignaciones no exceda el total
	totalAllocated := float64(0)
	for _, allocation := range req.Allocations {
		totalAllocated += allocation.AllocatedAmount
	}

	if totalAllocated > req.TotalAmount {
		return nil, apperrors.ErrBudgetAllocationsExceed
	}

	// Verificar que todas las categorías existen
	for _, allocation := range req.Allocations {
		category, err := uc.categoryRepo.GetByID(allocation.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("%w: category_id %d", apperrors.ErrCategoryNotFound, allocation.CategoryID)
		}

		// Verificar que la categoría pertenece al usuario o es del sistema
		if !category.IsSystemCategory() && category.UserID != nil && *category.UserID != userID {
			return nil, fmt.Errorf("%w: category_id %d does not belong to user", apperrors.ErrPermissionDenied, allocation.CategoryID)
		}
	}

	// Crear el presupuesto
	budget := &entity.Budget{
		UserID:          userID,
		Year:            req.Year,
		Month:           req.Month,
		TotalAmount:     req.TotalAmount,
		SpentAmount:     0,
		RemainingAmount: req.TotalAmount,
		IsActive:        true,
		AutoCreateNext:  true,
	}

	if err := uc.budgetRepo.Create(budget); err != nil {
		return nil, err
	}

	// Crear las asignaciones
	remainingDays := budget.GetRemainingDays()
	for _, allocReq := range req.Allocations {
		allocation := &entity.BudgetAllocation{
			BudgetID:        budget.ID,
			CategoryID:      allocReq.CategoryID,
			AllocatedAmount: allocReq.AllocatedAmount,
			SpentAmount:     0,
			RemainingAmount: allocReq.AllocatedAmount,
			AlertThreshold:  allocReq.AlertThreshold,
			IsOverBudget:    false,
		}

		// Calcular límite diario
		allocation.CalculateDailyLimit(remainingDays)

		if err := uc.budgetRepo.CreateAllocation(allocation); err != nil {
			return nil, err
		}
	}

	// Recargar el presupuesto con las asignaciones
	budgetWithAllocations, err := uc.budgetRepo.GetByIDWithAllocations(budget.ID)
	if err != nil {
		return nil, err
	}

	return uc.mapBudgetToSummaryResponse(budgetWithAllocations), nil
}

// GetCurrentBudget obtiene el presupuesto del mes actual
func (uc *BudgetUseCase) GetCurrentBudget(userID uint) (*dto.BudgetSummaryResponse, error) {
	now := time.Now()
	budget, err := uc.budgetRepo.GetByUserAndMonth(userID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, apperrors.ErrBudgetNotFound
	}

	// Recalcular límites diarios si es necesario
	if err := uc.updateDailyLimits(budget); err != nil {
		return nil, err
	}

	return uc.mapBudgetToSummaryResponse(budget), nil
}

// GetBudgetByMonth obtiene un presupuesto específico por mes
func (uc *BudgetUseCase) GetBudgetByMonth(userID uint, year, month int) (*dto.BudgetSummaryResponse, error) {
	budget, err := uc.budgetRepo.GetByUserAndMonth(userID, year, month)
	if err != nil {
		return nil, apperrors.ErrBudgetNotFound
	}

	return uc.mapBudgetToSummaryResponse(budget), nil
}

// UpdateBudget actualiza un presupuesto existente
func (uc *BudgetUseCase) UpdateBudget(userID, budgetID uint, req *dto.UpdateBudgetRequest) (*dto.BudgetSummaryResponse, error) {
	budget, err := uc.budgetRepo.GetByIDWithAllocations(budgetID)
	if err != nil {
		return nil, apperrors.ErrBudgetNotFound
	}

	// Verificar que pertenece al usuario
	if budget.UserID != userID {
		return nil, apperrors.ErrBudgetNotFound
	}

	// Actualizar monto total si se proporciona
	if req.TotalAmount != nil {
		// Validar que el nuevo total no sea menor que lo ya gastado
		if *req.TotalAmount < budget.SpentAmount {
			return nil, apperrors.ErrInvalidTotalAmount
		}

		budget.TotalAmount = *req.TotalAmount
		budget.CalculateRemainingAmount()
	}

	// Actualizar configuración
	if req.AutoCreateNext != nil {
		budget.AutoCreateNext = *req.AutoCreateNext
	}

	// Actualizar asignaciones si se proporcionan
	if req.Allocations != nil {
		for _, allocReq := range req.Allocations {
			allocation, err := uc.budgetRepo.GetAllocationByID(allocReq.ID)
			if err != nil {
				continue // Skip si no existe
			}

			// Verificar que pertenece al presupuesto
			if allocation.BudgetID != budget.ID {
				continue
			}

			// Actualizar campos
			if allocReq.AllocatedAmount != nil {
				if *allocReq.AllocatedAmount < allocation.SpentAmount {
					return nil, fmt.Errorf("%w: for category cannot be less than already spent (%.2f)", apperrors.ErrBudgetAllocationsExceed, allocation.SpentAmount)
				}
				allocation.AllocatedAmount = *allocReq.AllocatedAmount
				allocation.CalculateRemainingAmount()
			}

			if allocReq.AlertThreshold != nil {
				allocation.AlertThreshold = *allocReq.AlertThreshold
			}

			// Recalcular límite diario
			remainingDays := budget.GetRemainingDays()
			allocation.CalculateDailyLimit(remainingDays)

			if err := uc.budgetRepo.UpdateAllocation(allocation); err != nil {
				return nil, err
			}
		}
	}

	if err := uc.budgetRepo.Update(budget); err != nil {
		return nil, err
	}

	// Recargar con cambios
	updatedBudget, err := uc.budgetRepo.GetByIDWithAllocations(budgetID)
	if err != nil {
		return nil, err
	}

	return uc.mapBudgetToSummaryResponse(updatedBudget), nil
}

// GetDashboard obtiene el dashboard principal con información relevante
func (uc *BudgetUseCase) GetDashboard(userID uint) (*dto.BudgetDashboardResponse, error) {
	// Obtener presupuesto actual
	currentBudget, err := uc.GetCurrentBudget(userID)
	if err != nil {
		// Si no hay presupuesto actual, retornar dashboard vacío
		return &dto.BudgetDashboardResponse{
			CurrentBudget:  nil,
			TodayExpenses:  []dto.ExpenseSummaryResponse{},
			TodayTotal:     0,
			WeekTotal:      0,
			MonthTotal:     0,
			CategoryAlerts: []dto.AllocationAlert{},
			QuickStats:     dto.BudgetQuickStats{},
		}, nil
	}

	// Obtener gastos de hoy
	today := time.Now()
	todayExpenses, err := uc.expenseRepo.GetByUserAndDateRange(userID, &today, &today)
	if err != nil {
		todayExpenses = []*entity.Expense{}
	}

	// Calcular totales
	todayTotal := uc.calculateTotalAmount(todayExpenses)

	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	weekExpenses, _ := uc.expenseRepo.GetByUserAndDateRange(userID, &weekStart, &today)
	weekTotal := uc.calculateTotalAmount(weekExpenses)

	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	monthExpenses, _ := uc.expenseRepo.GetByUserAndDateRange(userID, &monthStart, &today)
	monthTotal := uc.calculateTotalAmount(monthExpenses)

	// Generar alertas por categoría
	alerts := uc.generateCategoryAlerts(currentBudget.Allocations)

	// Generar estadísticas rápidas
	quickStats := uc.generateQuickStats(currentBudget, monthExpenses)

	// Mapear gastos de hoy
	todayExpensesResponse := make([]dto.ExpenseSummaryResponse, len(todayExpenses))
	for i, expense := range todayExpenses {
		todayExpensesResponse[i] = uc.mapExpenseToSummaryResponse(expense)
	}

	return &dto.BudgetDashboardResponse{
		CurrentBudget:  currentBudget,
		TodayExpenses:  todayExpensesResponse,
		TodayTotal:     todayTotal,
		WeekTotal:      weekTotal,
		MonthTotal:     monthTotal,
		CategoryAlerts: alerts,
		QuickStats:     quickStats,
	}, nil
}

// ProcessDailyRollover procesa el rollover diario de saldos no gastados
func (uc *BudgetUseCase) ProcessDailyRollover(userID uint) error {
	// Obtener presupuesto actual
	now := time.Now()
	budget, err := uc.budgetRepo.GetByUserAndMonth(userID, now.Year(), int(now.Month()))
	if err != nil {
		return err // No hay presupuesto activo
	}

	yesterday := now.AddDate(0, 0, -1)

	// Para cada asignación, calcular el rollover
	for _, allocation := range budget.Allocations {
		// Obtener gastos de ayer en esta categoría
		yesterdayExpenses, err := uc.expenseRepo.GetByUserCategoryAndDate(userID, allocation.CategoryID, &yesterday)
		if err != nil {
			continue
		}

		yesterdaySpent := uc.calculateTotalAmount(yesterdayExpenses)
		unspentAmount := allocation.CurrentDailyLimit - yesterdaySpent

		// Si sobró dinero, agregarlo al límite de hoy
		if unspentAmount > 0 {
			allocation.AddRollover(unspentAmount)
			uc.budgetRepo.UpdateAllocation(&allocation)
		}

		// Recalcular límite diario base
		remainingDays := budget.GetRemainingDays()
		allocation.CalculateDailyLimit(remainingDays)
		uc.budgetRepo.UpdateAllocation(&allocation)
	}

	return nil
}

// Helper methods

func (uc *BudgetUseCase) updateDailyLimits(budget *entity.Budget) error {
	remainingDays := budget.GetRemainingDays()

	for _, allocation := range budget.Allocations {
		allocation.CalculateDailyLimit(remainingDays)
		if err := uc.budgetRepo.UpdateAllocation(&allocation); err != nil {
			return err
		}
	}

	return nil
}

func (uc *BudgetUseCase) mapBudgetToSummaryResponse(budget *entity.Budget) *dto.BudgetSummaryResponse {
	allocations := make([]dto.AllocationSummaryResponse, len(budget.Allocations))

	for i, allocation := range budget.Allocations {
		allocations[i] = dto.AllocationSummaryResponse{
			ID:                allocation.ID,
			Category:          uc.mapCategoryToSummaryResponse(&allocation.Category),
			AllocatedAmount:   allocation.AllocatedAmount,
			SpentAmount:       allocation.SpentAmount,
			RemainingAmount:   allocation.RemainingAmount,
			ProgressPercent:   allocation.GetProgressPercentage(),
			DailyLimit:        allocation.DailyLimit,
			CurrentDailyLimit: allocation.CurrentDailyLimit,
			AlertThreshold:    allocation.AlertThreshold,
			IsOverBudget:      allocation.IsOverBudgetCheck(),
			ShouldAlert:       allocation.ShouldAlert(),
			AllocationPercent: allocation.GetAllocationPercentage(budget.TotalAmount),
		}
	}

	return &dto.BudgetSummaryResponse{
		ID:              budget.ID,
		Year:            budget.Year,
		Month:           budget.Month,
		PeriodString:    budget.GetPeriodString(),
		TotalAmount:     budget.TotalAmount,
		SpentAmount:     budget.SpentAmount,
		RemainingAmount: budget.RemainingAmount,
		ProgressPercent: budget.GetProgressPercentage(),
		RemainingDays:   budget.GetRemainingDays(),
		IsActive:        budget.IsActive,
		IsCurrentMonth:  budget.IsCurrentMonth(),
		Allocations:     allocations,
	}
}

func (uc *BudgetUseCase) mapCategoryToSummaryResponse(category *entity.Category) dto.CategorySummaryResponse {
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

func (uc *BudgetUseCase) mapExpenseToSummaryResponse(expense *entity.Expense) dto.ExpenseSummaryResponse {
	return dto.ExpenseSummaryResponse{
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

func (uc *BudgetUseCase) calculateTotalAmount(expenses []*entity.Expense) float64 {
	total := float64(0)
	for _, expense := range expenses {
		if expense.IsConfirmed() {
			total += expense.Amount
		}
	}
	return total
}

func (uc *BudgetUseCase) generateCategoryAlerts(allocations []dto.AllocationSummaryResponse) []dto.AllocationAlert {
	alerts := []dto.AllocationAlert{}

	for _, allocation := range allocations {
		if allocation.ShouldAlert || allocation.IsOverBudget {
			alertType := "warning"
			message := fmt.Sprintf("Has gastado %.1f%% de tu presupuesto en %s",
				allocation.ProgressPercent, allocation.Category.Name)

			if allocation.IsOverBudget {
				alertType = "danger"
				message = fmt.Sprintf("Te has excedido en %s por $%.2f",
					allocation.Category.Name, allocation.SpentAmount-allocation.AllocatedAmount)
			} else if allocation.ProgressPercent >= 90 {
				alertType = "danger"
				message = fmt.Sprintf("Casi agotas tu presupuesto en %s", allocation.Category.Name)
			}

			alerts = append(alerts, dto.AllocationAlert{
				CategoryName:    allocation.Category.Name,
				CategoryIcon:    allocation.Category.Icon,
				AllocatedAmount: allocation.AllocatedAmount,
				SpentAmount:     allocation.SpentAmount,
				ProgressPercent: allocation.ProgressPercent,
				AlertType:       alertType,
				Message:         message,
			})
		}
	}

	return alerts
}

func (uc *BudgetUseCase) generateQuickStats(budget *dto.BudgetSummaryResponse, monthExpenses []*entity.Expense) dto.BudgetQuickStats {
	// Calcular estadísticas básicas
	daysInMonth := time.Date(budget.Year, time.Month(budget.Month+1), 0, 0, 0, 0, 0, time.UTC).Day()
	daysPassed := time.Now().Day()

	averageDaily := float64(0)
	if daysPassed > 0 {
		averageDaily = budget.SpentAmount / float64(daysPassed)
	}

	recommendedDaily := float64(0)
	if budget.RemainingDays > 0 {
		recommendedDaily = budget.RemainingAmount / float64(budget.RemainingDays)
	}

	categoriesOnTrack := 0
	categoriesOverBudget := 0

	for _, allocation := range budget.Allocations {
		if allocation.IsOverBudget {
			categoriesOverBudget++
		} else if allocation.ProgressPercent <= 80 { // Consideramos "on track" si está por debajo del 80%
			categoriesOnTrack++
		}
	}

	return dto.BudgetQuickStats{
		DaysUntilPayday:      daysInMonth - daysPassed,
		AverageDailySpent:    averageDaily,
		RecommendedDaily:     recommendedDaily,
		TotalCategories:      len(budget.Allocations),
		CategoriesOnTrack:    categoriesOnTrack,
		CategoriesOverBudget: categoriesOverBudget,
	}
}
