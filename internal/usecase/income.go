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

// IncomeUseCase contiene la lógica de negocio para ingresos
type IncomeUseCase struct {
	incomeRepo repo.IncomeRepo
	userRepo   repo.UserRepo
	logger     zerolog.Logger
}

// NewIncomeUseCase crea una nueva instancia de IncomeUseCase
func NewIncomeUseCase(incomeRepo repo.IncomeRepo, userRepo repo.UserRepo) *IncomeUseCase {
	return &IncomeUseCase{
		incomeRepo: incomeRepo,
		userRepo:   userRepo,
		logger:     logger.Get().With().Str("usecase", "Income").Logger(),
	}
}

// CreateIncome crea un nuevo ingreso
func (uc *IncomeUseCase) CreateIncome(userID uint, req *dto.CreateIncomeRequest) (*dto.IncomeResponse, error) {
	log := uc.logger.With().Uint("user_id", userID).Logger()
	log.Debug().Msg("CreateIncome started")

	// Verificar que el usuario existe
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		log.Warn().Err(err).Msg("User not found")
		return nil, errors.New("user not found")
	}

	if !user.IsAccountActive() {
		log.Warn().Msg("User account is not active")
		return nil, errors.New("user account is not active")
	}

	// Parsear fecha
	incomeDate, err := uc.parseDate(req.Date)
	if err != nil {
		log.Warn().Err(err).Str("date", req.Date).Msg("Invalid date")
		return nil, fmt.Errorf("invalid date format: %s", req.Date)
	}

	// Crear el ingreso
	income := &entity.Income{
		UserID:      userID,
		Amount:      req.Amount,
		Description: req.Description,
		Source:      entity.IncomeSource(req.Source),
		Date:        incomeDate,
		Notes:       req.Notes,
		Currency:    user.Currency,
		TaxDeducted: req.TaxDeducted,
		IsRecurring: req.IsRecurring,
	}

	// Configurar frecuencia si es recurrente
	if req.IsRecurring && req.Frequency != nil {
		frequency := entity.IncomeFrequency(*req.Frequency)
		income.Frequency = &frequency
	}

	// Configurar fecha de finalización si se proporciona
	if req.EndDate != nil && *req.EndDate != "" {
		endDate, err := uc.parseDate(*req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end date format: %s", *req.EndDate)
		}
		income.EndDate = &endDate
		income.RecurringUntil = &endDate
	}

	log.Debug().Interface("income", income).Msg("Creating income in database")
	if err := uc.incomeRepo.Create(income); err != nil {
		log.Error().Err(err).Msg("Error creating income in DB")
		return nil, fmt.Errorf("error creating income: %v", err)
	}

	log.Info().Uint("income_id", income.ID).Msg("Income created in DB successfully")
	return uc.mapIncomeToResponse(income), nil
}

// GetIncomes obtiene ingresos con filtros
func (uc *IncomeUseCase) GetIncomes(userID uint, startDate, endDate *time.Time, source *entity.IncomeSource, limit, offset int) ([]*dto.IncomeSummaryResponse, error) {
	var incomes []*entity.Income
	var err error

	if source != nil {
		incomes, err = uc.incomeRepo.GetByUserAndSource(userID, *source)
	} else {
		incomes, err = uc.incomeRepo.GetByUserAndDateRange(userID, startDate, endDate)
	}

	if err != nil {
		return nil, err
	}

	// Aplicar paginación manual
	if offset >= len(incomes) {
		return []*dto.IncomeSummaryResponse{}, nil
	}

	end := offset + limit
	if end > len(incomes) {
		end = len(incomes)
	}

	paginatedIncomes := incomes[offset:end]

	// Mapear a DTOs
	response := make([]*dto.IncomeSummaryResponse, len(paginatedIncomes))
	for i, income := range paginatedIncomes {
		response[i] = uc.mapIncomeToSummaryResponse(income)
	}

	return response, nil
}

// GetIncomeByID obtiene un ingreso por ID
func (uc *IncomeUseCase) GetIncomeByID(userID, incomeID uint) (*dto.IncomeResponse, error) {
	income, err := uc.incomeRepo.GetByID(incomeID)
	if err != nil {
		return nil, errors.New("income not found")
	}

	// Verificar que pertenece al usuario
	if income.UserID != userID {
		return nil, errors.New("income not found")
	}

	return uc.mapIncomeToResponse(income), nil
}

// UpdateIncome actualiza un ingreso existente
func (uc *IncomeUseCase) UpdateIncome(userID, incomeID uint, req *dto.UpdateIncomeRequest) (*dto.IncomeResponse, error) {
	income, err := uc.incomeRepo.GetByID(incomeID)
	if err != nil {
		return nil, errors.New("income not found")
	}

	// Verificar que pertenece al usuario
	if income.UserID != userID {
		return nil, errors.New("income not found")
	}

	// Verificar que puede ser modificado
	if !income.CanBeModified() {
		return nil, errors.New("income cannot be modified")
	}

	// Actualizar campos
	if req.Amount != nil {
		income.Amount = *req.Amount
	}

	if req.Description != "" {
		income.Description = req.Description
	}

	if req.Source != "" {
		income.Source = entity.IncomeSource(req.Source)
	}

	if req.Date != "" {
		incomeDate, err := uc.parseDate(req.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %s", req.Date)
		}
		income.Date = incomeDate
	}

	if req.Notes != "" {
		income.Notes = req.Notes
	}

	if req.TaxDeducted != nil {
		income.TaxDeducted = *req.TaxDeducted
	}

	if req.IsRecurring != nil {
		income.IsRecurring = *req.IsRecurring
	}

	if req.Frequency != nil {
		if *req.Frequency == "" {
			income.Frequency = nil
		} else {
			frequency := entity.IncomeFrequency(*req.Frequency)
			income.Frequency = &frequency
		}
	}

	if req.EndDate != nil {
		if *req.EndDate == "" {
			income.EndDate = nil
			income.RecurringUntil = nil
		} else {
			endDate, err := uc.parseDate(*req.EndDate)
			if err != nil {
				return nil, fmt.Errorf("invalid end date format: %s", *req.EndDate)
			}
			income.EndDate = &endDate
			income.RecurringUntil = &endDate
		}
	}

	if err := uc.incomeRepo.Update(income); err != nil {
		return nil, err
	}

	// Recargar con relaciones
	updatedIncome, err := uc.incomeRepo.GetByID(income.ID)
	if err != nil {
		return nil, err
	}

	return uc.mapIncomeToResponse(updatedIncome), nil
}

// DeleteIncome elimina un ingreso
func (uc *IncomeUseCase) DeleteIncome(userID, incomeID uint) error {
	income, err := uc.incomeRepo.GetByID(incomeID)
	if err != nil {
		return errors.New("income not found")
	}

	// Verificar que pertenece al usuario
	if income.UserID != userID {
		return errors.New("income not found")
	}

	// Verificar que puede ser eliminado
	if !income.CanBeDeleted() {
		return errors.New("income cannot be deleted")
	}

	return uc.incomeRepo.Delete(incomeID)
}

// GetIncomeStats obtiene estadísticas de ingresos
func (uc *IncomeUseCase) GetIncomeStats(userID uint, year *int) (*dto.IncomeStatsResponse, error) {
	currentYear := time.Now().Year()
	if year == nil {
		year = &currentYear
	}

	// Obtener ingresos del año
	incomes, err := uc.incomeRepo.GetIncomeByYear(userID, *year)
	if err != nil {
		return nil, err
	}

	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Calcular estadísticas
	stats := &dto.IncomeStatsResponse{
		Currency: user.Currency,
		Period:   fmt.Sprintf("%d", *year),
	}

	// Total de ingresos
	totalIncome := float64(0)
	for _, income := range incomes {
		totalIncome += income.Amount
	}
	stats.TotalIncome = totalIncome
	stats.FormattedTotalIncome = entity.FormatCurrency(totalIncome, user.Currency)

	// Promedio mensual
	monthlyAvg := totalIncome / 12
	stats.MonthlyAverage = monthlyAvg
	stats.FormattedMonthlyAverage = entity.FormatCurrency(monthlyAvg, user.Currency)

	// Ingresos por fuente
	stats.IncomeBySource = uc.calculateIncomeBySource(incomes, user.Currency)

	// Ingresos mensuales
	stats.MonthlyIncome = uc.calculateMonthlyIncome(incomes, *year, user.Currency)

	// Ingresos recurrentes
	recurringIncomes, err := uc.incomeRepo.GetRecurringIncomes(userID)
	if err == nil {
		for _, income := range recurringIncomes {
			stats.RecurringIncome = append(stats.RecurringIncome, *uc.mapIncomeToSummaryResponse(income))
		}
	}

	return stats, nil
}

// GetRecentIncomes obtiene ingresos recientes
func (uc *IncomeUseCase) GetRecentIncomes(userID uint, limit int) ([]*dto.IncomeSummaryResponse, error) {
	incomes, err := uc.incomeRepo.GetRecentIncomes(userID, limit)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.IncomeSummaryResponse, len(incomes))
	for i, income := range incomes {
		response[i] = uc.mapIncomeToSummaryResponse(income)
	}

	return response, nil
}

// ProcessRecurringIncomes procesa ingresos recurrentes pendientes
func (uc *IncomeUseCase) ProcessRecurringIncomes(userID uint) (*dto.RecurringIncomeProcessResponse, error) {
	log := uc.logger.With().Uint("user_id", userID).Logger()

	pendingIncomes, err := uc.incomeRepo.GetPendingRecurringIncomes(userID)
	if err != nil {
		return nil, err
	}

	processedIncomes := make([]dto.IncomeSummaryResponse, 0)
	processedCount := 0

	for _, income := range pendingIncomes {
		// Crear nuevo ingreso basado en el recurrente
		newIncome := &entity.Income{
			UserID:      income.UserID,
			Amount:      income.Amount,
			Description: income.Description,
			Source:      income.Source,
			Date:        *income.NextDate,
			Notes:       income.Notes + " (Generado automáticamente)",
			Currency:    income.Currency,
			TaxDeducted: income.TaxDeducted,
			IsRecurring: false, // El nuevo ingreso no es recurrente
		}

		if err := uc.incomeRepo.Create(newIncome); err != nil {
			log.Error().Err(err).Uint("recurring_income_id", income.ID).Msg("Error creating recurring income")
			continue
		}

		// Actualizar la próxima fecha del ingreso recurrente
		nextDate := income.CalculateNextDate()
		if err := uc.incomeRepo.UpdateNextRecurringDate(income.ID, nextDate); err != nil {
			log.Error().Err(err).Uint("recurring_income_id", income.ID).Msg("Error updating next recurring date")
		}

		processedIncomes = append(processedIncomes, *uc.mapIncomeToSummaryResponse(newIncome))
		processedCount++
	}

	return &dto.RecurringIncomeProcessResponse{
		ProcessedCount:   processedCount,
		ProcessedIncomes: processedIncomes,
		Message:          fmt.Sprintf("Se procesaron %d ingresos recurrentes", processedCount),
	}, nil
}

// Helper methods

func (uc *IncomeUseCase) parseDate(dateStr string) (time.Time, error) {
	dateFormats := []string{
		time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02T15:04:05.999999", // Formato con microsegundos (Flutter)
		"2006-01-02T15:04:05",        // ISO sin timezone
		"2006-01-02",                 // Solo fecha
	}

	for _, format := range dateFormats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, errors.New("invalid date format")
}

func (uc *IncomeUseCase) mapIncomeToResponse(income *entity.Income) *dto.IncomeResponse {
	response := &dto.IncomeResponse{
		ID:                 income.ID,
		Amount:             income.Amount,
		FormattedAmount:    income.GetFormattedAmount(),
		NetAmount:          income.NetAmount,
		FormattedNetAmount: income.GetFormattedNetAmount(),
		Description:        income.Description,
		Source:             string(income.Source),
		SourceDisplayName:  income.GetSourceDisplayName(),
		Date:               income.Date.Format(time.RFC3339),
		Notes:              income.Notes,
		Currency:           income.Currency,
		TaxDeducted:        income.TaxDeducted,
		IsRecurring:        income.IsRecurring,
		CanBeModified:      income.CanBeModified(),
		CanBeDeleted:       income.CanBeDeleted(),
		IsFuture:           income.IsFutureIncome(),
		IsActive:           income.IsActive(),
		CreatedAt:          income.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          income.UpdatedAt.Format(time.RFC3339),
	}

	if income.Frequency != nil {
		freq := string(*income.Frequency)
		freqDisplay := income.GetFrequencyDisplayName()
		response.Frequency = &freq
		response.FrequencyDisplayName = &freqDisplay
	}

	if income.NextDate != nil {
		nextDate := income.NextDate.Format(time.RFC3339)
		response.NextDate = &nextDate
	}

	if income.EndDate != nil {
		endDate := income.EndDate.Format(time.RFC3339)
		response.EndDate = &endDate
	}

	return response
}

func (uc *IncomeUseCase) mapIncomeToSummaryResponse(income *entity.Income) *dto.IncomeSummaryResponse {
	return &dto.IncomeSummaryResponse{
		ID:                income.ID,
		Amount:            income.Amount,
		FormattedAmount:   income.GetFormattedAmount(),
		Description:       income.Description,
		Source:            string(income.Source),
		SourceDisplayName: income.GetSourceDisplayName(),
		Date:              income.Date.Format(time.RFC3339),
		Currency:          income.Currency,
		IsRecurring:       income.IsRecurring,
		CreatedAt:         income.CreatedAt.Format(time.RFC3339),
	}
}

func (uc *IncomeUseCase) calculateIncomeBySource(incomes []*entity.Income, currency string) []dto.IncomeBySourceResponse {
	sourceMap := make(map[entity.IncomeSource]float64)
	total := float64(0)

	for _, income := range incomes {
		sourceMap[income.Source] += income.Amount
		total += income.Amount
	}

	var result []dto.IncomeBySourceResponse
	for source, amount := range sourceMap {
		percentage := float64(0)
		if total > 0 {
			percentage = (amount / total) * 100
		}

		result = append(result, dto.IncomeBySourceResponse{
			Source:            string(source),
			SourceDisplayName: (&entity.Income{Source: source}).GetSourceDisplayName(),
			TotalAmount:       amount,
			FormattedAmount:   entity.FormatCurrency(amount, currency),
			Percentage:        percentage,
		})
	}

	return result
}

func (uc *IncomeUseCase) calculateMonthlyIncome(incomes []*entity.Income, year int, currency string) []dto.MonthlyIncomeResponse {
	monthlyMap := make(map[int]float64)
	monthlyCount := make(map[int]int)

	for _, income := range incomes {
		if income.Date.Year() == year {
			month := int(income.Date.Month())
			monthlyMap[month] += income.Amount
			monthlyCount[month]++
		}
	}

	var result []dto.MonthlyIncomeResponse
	months := []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
		"Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}

	for month := 1; month <= 12; month++ {
		amount := monthlyMap[month]
		count := monthlyCount[month]

		result = append(result, dto.MonthlyIncomeResponse{
			Year:            year,
			Month:           month,
			MonthName:       months[month],
			TotalAmount:     amount,
			FormattedAmount: entity.FormatCurrency(amount, currency),
			Count:           count,
		})
	}

	return result
}
