package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/internal/usecase/webapi"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// BankNotificationPatternUseCase contiene la lógica de negocio para patrones de notificación bancaria
type BankNotificationPatternUseCase struct {
	patternRepo       repo.BankNotificationPatternRepo
	bankAccountRepo   repo.BankAccountRepo
	userRepo          repo.UserRepo
	transactionRepo   repo.TransactionRepo
	expenseRepo       repo.ExpenseRepo
	incomeRepo        repo.IncomeRepo
	budgetRepo        repo.BudgetRepo
	categoryRepo      repo.CategoryRepo
	slugStatsRepo     repo.BudgetSuggestionSlugStatsRepo
	suggestionJobRepo repo.BudgetSuggestionJobRepo
	aiService         *webapi.AIServiceWithFallback
}

// NewBankNotificationPatternUseCase crea una nueva instancia de BankNotificationPatternUseCase
func NewBankNotificationPatternUseCase(
	patternRepo repo.BankNotificationPatternRepo,
	bankAccountRepo repo.BankAccountRepo,
	userRepo repo.UserRepo,
	transactionRepo repo.TransactionRepo,
	expenseRepo repo.ExpenseRepo,
	incomeRepo repo.IncomeRepo,
	budgetRepo repo.BudgetRepo,
	categoryRepo repo.CategoryRepo,
	slugStatsRepo repo.BudgetSuggestionSlugStatsRepo,
	suggestionJobRepo repo.BudgetSuggestionJobRepo,
	aiService *webapi.AIServiceWithFallback,
) *BankNotificationPatternUseCase {
	return &BankNotificationPatternUseCase{
		patternRepo:       patternRepo,
		bankAccountRepo:   bankAccountRepo,
		userRepo:          userRepo,
		transactionRepo:   transactionRepo,
		expenseRepo:       expenseRepo,
		incomeRepo:        incomeRepo,
		budgetRepo:        budgetRepo,
		categoryRepo:      categoryRepo,
		slugStatsRepo:     slugStatsRepo,
		suggestionJobRepo: suggestionJobRepo,
		aiService:         aiService,
	}
}

// CreatePattern crea una nueva configuración de notificación bancaria
func (uc *BankNotificationPatternUseCase) CreatePattern(userID uint, req *dto.CreateBankNotificationPatternRequest) (*dto.BankNotificationPatternResponse, error) {
	// Verificar que la cuenta bancaria existe y pertenece al usuario
	bankAccount, err := uc.bankAccountRepo.GetByID(req.BankAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil {
		return nil, errors.New("bank account not found")
	}
	if bankAccount.UserID != userID {
		return nil, errors.New("unauthorized access to bank account")
	}

	// Crear la entidad configuración
	pattern := &entity.BankNotificationPattern{
		UserID:              userID,
		BankAccountID:       req.BankAccountID,
		Name:                req.Name,
		Description:         req.Description,
		Channel:             req.Channel,
		Status:              entity.NotificationPatternStatusActive,
		ExampleMessage:      req.ExampleMessage,
		RequiresValidation:  req.RequiresValidation,
		ConfidenceThreshold: req.ConfidenceThreshold,
		AutoApprove:         req.AutoApprove,
		Priority:            req.Priority,
		IsDefault:           req.IsDefault,
	}

	// Establecer tags
	if len(req.Tags) > 0 {
		if err := pattern.SetTags(req.Tags); err != nil {
			return nil, fmt.Errorf("failed to set tags: %w", err)
		}
	}
	if req.Metadata != nil {
		if err := pattern.SetMetadata(req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	// Si se está marcando como por defecto, desactivar otros por defecto
	if req.IsDefault {
		if err := uc.unsetOtherDefaultPatterns(req.BankAccountID, req.Channel); err != nil {
			return nil, fmt.Errorf("failed to unset other default patterns: %w", err)
		}
	}

	// Crear la configuración
	if err := uc.patternRepo.Create(pattern); err != nil {
		return nil, fmt.Errorf("failed to create pattern: %w", err)
	}

	response := uc.toDTO(pattern)
	return response, nil
}

// GetPattern obtiene un patrón por ID
func (uc *BankNotificationPatternUseCase) GetPattern(userID, patternID uint) (*dto.BankNotificationPatternResponse, error) {
	pattern, err := uc.patternRepo.GetByID(patternID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}
	if pattern == nil {
		return nil, errors.New("pattern not found")
	}

	// Verificar que el patrón pertenece al usuario
	if pattern.UserID != userID {
		return nil, errors.New("unauthorized access to pattern")
	}

	response := uc.toDTO(pattern)
	return response, nil
}

// GetUserPatterns obtiene todos los patrones de un usuario
func (uc *BankNotificationPatternUseCase) GetUserPatterns(userID uint) ([]*dto.BankNotificationPatternResponse, error) {
	patterns, err := uc.patternRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user patterns: %w", err)
	}

	responses := make([]*dto.BankNotificationPatternResponse, len(patterns))
	for i, pattern := range patterns {
		responses[i] = uc.toDTO(pattern)
	}

	return responses, nil
}

// GetBankAccountPatterns obtiene patrones de una cuenta bancaria
func (uc *BankNotificationPatternUseCase) GetBankAccountPatterns(userID, bankAccountID uint, activeOnly bool) ([]*dto.BankNotificationPatternResponse, error) {
	// Verificar que la cuenta bancaria pertenece al usuario
	bankAccount, err := uc.bankAccountRepo.GetByID(bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil || bankAccount.UserID != userID {
		return nil, errors.New("unauthorized access to bank account")
	}

	var patterns []*entity.BankNotificationPattern
	if activeOnly {
		patterns, err = uc.patternRepo.GetActiveByBankAccountID(bankAccountID)
	} else {
		patterns, err = uc.patternRepo.GetByBankAccountID(bankAccountID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get bank account patterns: %w", err)
	}

	responses := make([]*dto.BankNotificationPatternResponse, len(patterns))
	for i, pattern := range patterns {
		responses[i] = uc.toDTO(pattern)
	}

	return responses, nil
}

// UpdatePattern actualiza una configuración existente
func (uc *BankNotificationPatternUseCase) UpdatePattern(userID, patternID uint, req *dto.UpdateBankNotificationPatternRequest) (*dto.BankNotificationPatternResponse, error) {
	// Obtener la configuración existente
	pattern, err := uc.patternRepo.GetByID(patternID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}
	if pattern == nil {
		return nil, errors.New("pattern not found")
	}
	if pattern.UserID != userID {
		return nil, errors.New("unauthorized access to pattern")
	}

	// Actualizar campos
	if req.Name != nil && *req.Name != "" {
		pattern.Name = *req.Name
	}
	if req.Description != nil && *req.Description != "" {
		pattern.Description = *req.Description
	}
	if req.ExampleMessage != nil && *req.ExampleMessage != "" {
		pattern.ExampleMessage = *req.ExampleMessage
	}
	if req.RequiresValidation != nil {
		pattern.RequiresValidation = *req.RequiresValidation
	}
	if req.ConfidenceThreshold != nil {
		pattern.ConfidenceThreshold = *req.ConfidenceThreshold
	}
	if req.AutoApprove != nil {
		pattern.AutoApprove = *req.AutoApprove
	}
	if req.Priority != nil {
		pattern.Priority = *req.Priority
	}
	if req.IsDefault != nil && *req.IsDefault != pattern.IsDefault {
		if *req.IsDefault {
			// Desactivar otros por defecto
			if err := uc.unsetOtherDefaultPatterns(pattern.BankAccountID, pattern.Channel); err != nil {
				return nil, fmt.Errorf("failed to unset other default patterns: %w", err)
			}
		}
		pattern.IsDefault = *req.IsDefault
	}

	// Actualizar tags si se proporcionan
	if req.Tags != nil {
		if err := pattern.SetTags(req.Tags); err != nil {
			return nil, fmt.Errorf("failed to set tags: %w", err)
		}
	}
	if req.Metadata != nil {
		if err := pattern.SetMetadata(req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	// Guardar cambios
	if err := uc.patternRepo.Update(pattern); err != nil {
		return nil, fmt.Errorf("failed to update pattern: %w", err)
	}

	response := uc.toDTO(pattern)
	return response, nil
}

// DeletePattern elimina un patrón
func (uc *BankNotificationPatternUseCase) DeletePattern(userID, patternID uint) error {
	// Verificar que el patrón existe y pertenece al usuario
	pattern, err := uc.patternRepo.GetByID(patternID)
	if err != nil {
		return fmt.Errorf("failed to get pattern: %w", err)
	}
	if pattern == nil {
		return errors.New("pattern not found")
	}
	if pattern.UserID != userID {
		return errors.New("unauthorized access to pattern")
	}

	// Eliminar el patrón
	if err := uc.patternRepo.Delete(patternID); err != nil {
		return fmt.Errorf("failed to delete pattern: %w", err)
	}

	return nil
}

// SetPatternStatus cambia el estado de un patrón
func (uc *BankNotificationPatternUseCase) SetPatternStatus(userID, patternID uint, status entity.NotificationPatternStatus) error {
	// Verificar que el patrón existe y pertenece al usuario
	pattern, err := uc.patternRepo.GetByID(patternID)
	if err != nil {
		return fmt.Errorf("failed to get pattern: %w", err)
	}
	if pattern == nil {
		return errors.New("pattern not found")
	}
	if pattern.UserID != userID {
		return errors.New("unauthorized access to pattern")
	}

	// Cambiar el estado
	if err := uc.patternRepo.SetStatus(patternID, status); err != nil {
		return fmt.Errorf("failed to set pattern status: %w", err)
	}

	return nil
}

// GetPatternStatistics obtiene estadísticas de patrones de un usuario
func (uc *BankNotificationPatternUseCase) GetPatternStatistics(userID uint) (*dto.PatternStatisticsResponse, error) {
	patterns, err := uc.patternRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get patterns: %w", err)
	}

	stats := &dto.PatternStatisticsResponse{
		TotalPatterns:    len(patterns),
		ActivePatterns:   0,
		LearningPatterns: 0,
		TotalMatches:     0,
		TotalSuccesses:   0,
	}

	for _, pattern := range patterns {
		switch pattern.Status {
		case entity.NotificationPatternStatusActive:
			stats.ActivePatterns++
		case entity.NotificationPatternStatusLearning:
			stats.LearningPatterns++
		}
		stats.TotalMatches += pattern.MatchCount
		stats.TotalSuccesses += pattern.SuccessCount
	}

	if stats.TotalMatches > 0 {
		stats.OverallSuccessRate = float64(stats.TotalSuccesses) / float64(stats.TotalMatches) * 100
	}

	return stats, nil
}

// unsetOtherDefaultPatterns desactiva otros patrones por defecto para la misma cuenta y canal
func (uc *BankNotificationPatternUseCase) unsetOtherDefaultPatterns(bankAccountID uint, channel entity.NotificationChannel) error {
	patterns, err := uc.patternRepo.GetByBankAccountID(bankAccountID)
	if err != nil {
		return err
	}

	for _, pattern := range patterns {
		if pattern.Channel == channel && pattern.IsDefault {
			if err := uc.patternRepo.SetDefault(pattern.ID, false); err != nil {
				return err
			}
		}
	}

	return nil
}

// toDTO convierte una entidad BankNotificationPattern a DTO de respuesta
func (uc *BankNotificationPatternUseCase) toDTO(pattern *entity.BankNotificationPattern) *dto.BankNotificationPatternResponse {
	return &dto.BankNotificationPatternResponse{
		ID:                  pattern.ID,
		BankAccountID:       pattern.BankAccountID,
		Name:                pattern.Name,
		Description:         pattern.Description,
		Channel:             pattern.Channel,
		Status:              pattern.Status,
		ExampleMessage:      pattern.ExampleMessage,
		RequiresValidation:  pattern.RequiresValidation,
		ConfidenceThreshold: pattern.ConfidenceThreshold,
		AutoApprove:         pattern.AutoApprove,
		MatchCount:          pattern.MatchCount,
		SuccessCount:        pattern.SuccessCount,
		SuccessRate:         pattern.SuccessRate,
		LastMatchedAt:       pattern.LastMatchedAt,
		Priority:            pattern.Priority,
		IsDefault:           pattern.IsDefault,
		Tags:                pattern.GetTags(),
		Metadata:            pattern.GetMetadata(),
		CreatedAt:           pattern.CreatedAt,
		UpdatedAt:           pattern.UpdatedAt,
	}
}

// ProcessNotificationWebhook procesa una notificación bancaria desde webhook usando IA
func (uc *BankNotificationPatternUseCase) ProcessNotificationWebhook(req dto.ProcessNotificationRequest) (*dto.ProcessedNotificationResponse, error) {
	// Buscar usuario por teléfono si no se proporciona UserID
	var userID uint
	if req.UserID > 0 {
		userID = req.UserID
	} else {
		// Buscar usuario por teléfono en cuentas bancarias
		bankAccount, err := uc.findBankAccountByPhone(req.Phone)
		if err != nil {
			return nil, apperrors.ErrNotFound.WithDetails("No se encontró cuenta bancaria asociada al teléfono")
		}
		userID = bankAccount.UserID
	}

	// Usar IA para procesar el mensaje directamente
	ctx := context.Background()
	return uc.ProcessSMSWithAI(ctx, userID, req.Message)
}

// ProcessSMSWithAI processes an SMS directly using AI without pattern matching.
// This is the new simplified flow that uses only AI with fallback support.
func (uc *BankNotificationPatternUseCase) ProcessSMSWithAI(ctx context.Context, userID uint, message string) (*dto.ProcessedNotificationResponse, error) {
	if uc.aiService == nil {
		return nil, apperrors.ErrInternal.WithDetails("Servicio de IA no está configurado")
	}

	// 1. Extract transaction data using AI (with automatic fallback)
	extraction, err := uc.aiService.ExtractTransactionFromSMS(ctx, message)
	if err != nil {
		return &dto.ProcessedNotificationResponse{
			Success:            false,
			TransactionCreated: false,
			Confidence:         0.0,
			Reason:             fmt.Sprintf("Error al procesar SMS con IA: %v", err),
		}, nil
	}

	// 2. Build response
	response := &dto.ProcessedNotificationResponse{
		Success:            extraction.Success,
		Confidence:         extraction.Confidence,
		PatternUsed:        uc.aiService.GetUsedService(),
		RequiresValidation: extraction.Confidence < 0.8,
		ExtractedData: map[string]interface{}{
			"amount":           extraction.Amount,
			"description":      extraction.Description,
			"merchant":         extraction.Merchant,
			"date":             extraction.Date,
			"transaction_type": extraction.TransactionType,
			"currency":         extraction.Currency,
		},
	}

	// 3. Determine if we should auto-create transaction
	if extraction.Success && extraction.Confidence >= 0.8 {
		// Create transaction automatically
		transaction, err := uc.createTransactionFromAIExtraction(userID, extraction, message)
		if err != nil {
			response.TransactionCreated = false
			response.RequiresValidation = true
			response.Reason = fmt.Sprintf("Error al crear transacción: %v", err)
		} else {
			response.TransactionCreated = true
			response.TransactionID = transaction.ID
			response.Amount = transaction.Amount
			response.Description = transaction.Description
		}
	} else {
		response.RequiresValidation = true
		if !extraction.Success {
			response.Reason = "No se pudo extraer información del SMS"
		} else {
			response.Reason = "Confianza insuficiente, requiere validación manual"
		}
	}

	return response, nil
}

// createTransactionFromAIExtraction creates an expense or income from AI extraction data
func (uc *BankNotificationPatternUseCase) createTransactionFromAIExtraction(
	userID uint,
	extraction *webapi.TransactionExtraction,
	rawMessage string,
) (*entity.Transaction, error) {
	// Parse date
	transactionDate := time.Now()
	if extraction.Date != "" {
		if parsedDate, err := uc.parseDate(extraction.Date); err == nil {
			transactionDate = parsedDate
		}
	}

	// Get description
	description := extraction.Description
	if description == "" && extraction.Merchant != "" {
		description = extraction.Merchant
	}
	if description == "" {
		description = "Transacción automática"
	}

	// Handle based on transaction type
	if extraction.TransactionType == "income" {
		// Create income
		income := &entity.Income{
			UserID:      userID,
			Amount:      extraction.Amount,
			Description: description,
			Source:      entity.IncomeSourceOther,
			Date:        transactionDate,
			Notes:       rawMessage,
			Currency:    extraction.Currency,
		}

		if income.Currency == "" {
			income.Currency = "COP"
		}

		if err := uc.incomeRepo.Create(income); err != nil {
			return nil, apperrors.ErrInternal.WithInternal(err).WithDetails("Error al crear ingreso")
		}

		// Return a transaction object for response compatibility
		return &entity.Transaction{
			ID:              income.ID,
			UserID:          userID,
			Type:            entity.TransactionTypeIncome,
			Amount:          income.Amount,
			Description:     income.Description,
			TransactionDate: income.Date,
		}, nil
	}

	// For expenses
	// Get current budget
	budget, err := uc.budgetRepo.GetCurrentBudget(userID)
	if err != nil || budget == nil {
		return nil, apperrors.ErrNotFound.WithDetails("No se encontró un presupuesto activo. Por favor crea un presupuesto primero.")
	}

	// Get default category (first available or "Otros")
	categories, err := uc.categoryRepo.GetAllAvailableForUser(userID)
	if err != nil || len(categories) == 0 {
		return nil, apperrors.ErrNotFound.WithDetails("No se encontraron categorías disponibles")
	}

	// Find "Otros" category or use first one
	var defaultCategory *entity.Category
	for _, cat := range categories {
		if cat.Name == "Otros" || cat.Name == "Other" {
			defaultCategory = cat
			break
		}
	}
	if defaultCategory == nil {
		defaultCategory = categories[0]
	}

	// Get or create allocation for this category
	allocation, err := uc.budgetRepo.GetAllocationByBudgetAndCategory(budget.ID, defaultCategory.ID)
	if err != nil || allocation == nil {
		// Create a default allocation if it doesn't exist
		allocation = &entity.BudgetAllocation{
			BudgetID:        budget.ID,
			CategoryID:      defaultCategory.ID,
			AllocatedAmount: 0, // No allocated amount for auto-created
			SpentAmount:     0,
			AlertThreshold:  0.8,
		}
		if err := uc.budgetRepo.CreateAllocation(allocation); err != nil {
			return nil, apperrors.ErrInternal.WithInternal(err).WithDetails("Error al crear asignación de presupuesto")
		}
	}

	// Create expense
	expense := &entity.Expense{
		UserID:       userID,
		BudgetID:     budget.ID,
		CategoryID:   defaultCategory.ID,
		AllocationID: allocation.ID,
		Amount:       extraction.Amount,
		Description:  description,
		Date:         transactionDate,
		Source:       entity.ExpenseSourceNotification,
		Status:       entity.ExpenseStatusPending, // Pending for user review
		Merchant:     extraction.Merchant,
		RawData:      rawMessage,
		Confidence:   extraction.Confidence,
		Currency:     extraction.Currency,
	}

	if expense.Currency == "" {
		expense.Currency = "COP"
	}

	if err := uc.expenseRepo.Create(expense); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err).WithDetails("Error al crear gasto")
	}

	// Update allocation spent amount
	_ = uc.budgetRepo.UpdateAllocationSpentAmount(allocation.ID)

	// Return a transaction object for response compatibility
	return &entity.Transaction{
		ID:              expense.ID,
		UserID:          userID,
		Type:            entity.TransactionTypeExpense,
		Amount:          expense.Amount,
		Description:     expense.Description,
		TransactionDate: expense.Date,
	}, nil
}

// parseExtractedData convierte los valores extraídos a sus tipos correctos
func (uc *BankNotificationPatternUseCase) parseExtractedData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	cleanedData := make(map[string]interface{})

	// Parse Amount (string to float64)
	if amountStr, ok := data["amount"].(string); ok {
		// Limpiar el string: quitar comas y espacios
		amountStr = strings.ReplaceAll(amountStr, ",", "")
		amountStr = strings.TrimSpace(amountStr)
		if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
			cleanedData["amount"] = amount
		}
	}

	// Copiar otros campos que ya son strings
	if dateStr, ok := data["date"].(string); ok {
		cleanedData["date"] = dateStr
	}
	if descStr, ok := data["description"].(string); ok {
		cleanedData["description"] = descStr
	}
	if merchStr, ok := data["merchant"].(string); ok {
		cleanedData["merchant"] = merchStr
	}

	return cleanedData
}

// createTransactionFromNotification crea una transacción a partir de los datos extraídos
func (uc *BankNotificationPatternUseCase) createTransactionFromNotification(
	userID uint,
	pattern *entity.BankNotificationPattern,
	data map[string]interface{},
	req dto.ProcessNotificationRequest,
) (*entity.Transaction, error) {
	// Extraer monto
	amount, ok := data["amount"].(float64)
	if !ok {
		return nil, apperrors.ErrInvalidRequest.WithDetails("No se pudo extraer el monto de la notificación")
	}

	// Extraer descripción
	description := "Transacción automática"
	if desc, ok := data["description"].(string); ok && desc != "" {
		description = desc
	}
	if merchant, ok := data["merchant"].(string); ok && merchant != "" {
		description = merchant
	}

	// Obtener cuenta bancaria (para validación)
	_, err := uc.bankAccountRepo.GetByID(pattern.BankAccountID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err).WithDetails("Error al obtener cuenta bancaria")
	}

	// Determinar tipo de transacción (por defecto gasto)
	transactionType := entity.TransactionTypeExpense

	// Crear transacción
	transaction := &entity.Transaction{
		UserID:           userID,
		AccountID:        1, // TODO: Obtener cuenta por defecto del usuario
		BankAccountID:    &pattern.BankAccountID,
		Type:             transactionType,
		Amount:           amount,
		Description:      description,
		Source:           entity.TransactionSourceNotification,
		ValidationStatus: entity.ValidationStatusAuto,
		RawNotification:  req.Message,
		AIConfidence:     uc.calculateConfidence(data),
		PatternID:        &pattern.ID,
		TransactionDate:  time.Now(),
	}

	// Extraer fecha si está disponible
	if dateStr, ok := data["date"].(string); ok {
		if parsedDate, err := uc.parseDate(dateStr); err == nil {
			transaction.TransactionDate = parsedDate
		}
	}

	// Guardar transacción
	if err := uc.transactionRepo.Create(transaction); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err).WithDetails("Error al crear transacción")
	}

	return transaction, nil
}

// findBankAccountByPhone busca una cuenta bancaria por teléfono de notificación
func (uc *BankNotificationPatternUseCase) findBankAccountByPhone(phone string) (*entity.BankAccount, error) {
	// Buscar cuentas que tengan este teléfono configurado para notificaciones
	accounts, err := uc.bankAccountRepo.GetByNotificationPhone(phone)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, apperrors.ErrNotFound.WithDetails("No se encontró cuenta bancaria para este teléfono")
	}

	// Retornar la primera cuenta encontrada
	// TODO: Mejorar lógica si hay múltiples cuentas con el mismo teléfono
	return accounts[0], nil
}

// calculateConfidence calcula la confianza basada en los datos extraídos
func (uc *BankNotificationPatternUseCase) calculateConfidence(data map[string]interface{}) float64 {
	confidence := 0.0
	totalFields := 0

	// Verificar campos extraídos
	if _, ok := data["amount"]; ok {
		confidence += 0.4 // Monto es crítico
		totalFields++
	}
	if _, ok := data["description"]; ok {
		confidence += 0.2
		totalFields++
	}
	if _, ok := data["merchant"]; ok {
		confidence += 0.2
		totalFields++
	}
	if _, ok := data["date"]; ok {
		confidence += 0.2
		totalFields++
	}

	if totalFields == 0 {
		return 0.0
	}

	return confidence
}

// parseDate intenta parsear una fecha de diferentes formatos
func (uc *BankNotificationPatternUseCase) parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"02/01/2006",
		"2006-01-02",
		"02-01-2006",
		"02/01/06",
		"2/1/2006",
		"2/1/06",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, errors.New("formato de fecha no reconocido")
}

// GetPendingNotifications obtiene notificaciones pendientes de validación
func (uc *BankNotificationPatternUseCase) GetPendingNotifications(userID uint, limit int) ([]*dto.PendingNotification, error) {
	// TODO: Implementar tabla de notificaciones pendientes
	// Por ahora retornamos una lista vacía
	return []*dto.PendingNotification{}, nil
}

// ProcessSMSBatchForSuggestions analyzes multiple SMS with pocas llamadas a la IA (lotes), no una por SMS.
// Solo se usan los SMS más recientes (cap bajo) para no hacer esperar minutos al usuario; el perfil de gasto suele verse bien con esa muestra.
func (uc *BankNotificationPatternUseCase) ProcessSMSBatchForSuggestions(ctx context.Context, userID uint, messages []dto.SMSMessageForAnalysis) (*dto.AnalyzeSMSBatchResponse, error) {
	const (
		maxSMSForSuggestions = 100 // muestra reciente; pocas llamadas a la IA
		smsPerChunk          = 25  // máx. 4 chunks con el cap anterior
		maxAIChunks          = 4   // cinturón: no más de 4 rondas aunque cambien constantes
		maxBodyRunes         = 220
		minConfidence        = 0.3
	)

	sortSMSMessagesNewestFirst(messages)
	if len(messages) > maxSMSForSuggestions {
		messages = messages[:maxSMSForSuggestions]
	}

	categories, err := uc.categoryRepo.GetAllAvailableForUser(userID)
	if err != nil || len(categories) == 0 {
		return &dto.AnalyzeSMSBatchResponse{
			Suggestions: dto.BudgetSuggestions{ByCategory: []dto.BudgetSuggestionCategory{}},
		}, nil
	}

	var otrosCategory *entity.Category
	for _, c := range categories {
		if c.Name == "Otros" || c.Name == "Other" {
			otrosCategory = c
			break
		}
	}
	if otrosCategory == nil {
		otrosCategory = categories[0]
	}

	// Solo SMS que parecen movimientos reales; no reinyectar todo el inbox (ofertas/moras/OTP).
	filtered := filterMessagesLikelyBankSMS(messages)

	type catAgg struct {
		total float64
		count int
	}
	byCatID := make(map[uint]*catAgg)
	slugHits := make(map[string]int)

	chunksDone := 0
	for chunkStart := 0; chunkStart < len(filtered) && chunksDone < maxAIChunks; chunkStart += smsPerChunk {
		chunkEnd := chunkStart + smsPerChunk
		if chunkEnd > len(filtered) {
			chunkEnd = len(filtered)
		}
		chunk := filtered[chunkStart:chunkEnd]
		chunksDone++
		var sb strings.Builder
		for i, msg := range chunk {
			lineNum := i + 1
			body := sanitizeSMSLine(msg.Body, maxBodyRunes)
			if body == "" {
				continue
			}
			sb.WriteString(fmt.Sprintf("[%d] %s\n", lineNum, body))
		}
		block := sb.String()
		if strings.TrimSpace(block) == "" {
			continue
		}

		resp, err := uc.aiService.ExtractBudgetLinesFromSMSChunk(ctx, block)
		if err != nil || resp == nil {
			continue
		}
		seenLine := make(map[int]bool)
		for _, line := range resp.Lines {
			if line.Line < 1 || seenLine[line.Line] {
				continue
			}
			seenLine[line.Line] = true
			if strings.ToLower(strings.TrimSpace(line.TransactionType)) != "expense" {
				continue
			}
			if line.Confidence < minConfidence {
				continue
			}
			if line.Amount <= 0 || line.Amount > 1e9 {
				continue
			}
			cat := resolveBudgetCategory(line.CategoryKey, categories, otrosCategory)
			if byCatID[cat.ID] == nil {
				byCatID[cat.ID] = &catAgg{}
			}
			byCatID[cat.ID].total += line.Amount
			byCatID[cat.ID].count++
			slugHits[normalizeSlugForAnalytics(line.CategoryKey)]++
		}
	}

	var totalExpense float64
	var expenseCount int
	byCategory := make([]dto.BudgetSuggestionCategory, 0, len(byCatID))
	for _, c := range categories {
		agg := byCatID[c.ID]
		if agg == nil || agg.total <= 0 {
			continue
		}
		totalExpense += agg.total
		expenseCount += agg.count
		byCategory = append(byCategory, dto.BudgetSuggestionCategory{
			CategoryID:   c.ID,
			CategoryName: c.Name,
			Total:        agg.total,
			Count:        agg.count,
		})
	}
	sort.Slice(byCategory, func(i, j int) bool {
		return byCategory[i].Total > byCategory[j].Total
	})

	if len(slugHits) > 0 && uc.slugStatsRepo != nil {
		countsCopy := make(map[string]int64, len(slugHits))
		for k, v := range slugHits {
			countsCopy[k] = int64(v)
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := uc.slugStatsRepo.AddHits(ctx, time.Now().UTC(), countsCopy); err != nil {
				log.Printf("budget_suggestion_slug_stats: %v", err)
			}
		}()
	}

	return &dto.AnalyzeSMSBatchResponse{
		Suggestions: dto.BudgetSuggestions{
			TotalExpense3m: totalExpense,
			ByCategory:     byCategory,
		},
	}, nil
}

// ProcessSMSBatchWithAI agrupa SMS en chunks, llama a la IA una vez por chunk y crea transacciones si confianza >= 0.8.
func (uc *BankNotificationPatternUseCase) ProcessSMSBatchWithAI(ctx context.Context, userID uint, messages []dto.SMSMessageForAnalysis) (*dto.ProcessSMSBatchWithAIResponse, error) {
	const (
		maxSMSInBatch     = 200
		smsPerChunk       = 20
		maxAIChunks       = 12
		maxBodyRunes      = 400
		minAutoConfidence = 0.8
	)

	totalIn := len(messages)
	out := &dto.ProcessSMSBatchWithAIResponse{
		TotalReceived: totalIn,
	}

	if uc.aiService == nil {
		return nil, apperrors.ErrInternal.WithDetails("Servicio de IA no está configurado")
	}

	if len(messages) == 0 {
		return out, nil
	}

	sortSMSMessagesNewestFirst(messages)
	toProcess := messages
	if len(toProcess) > maxSMSInBatch {
		toProcess = toProcess[:maxSMSInBatch]
	}

	filtered := filterMessagesLikelyBankSMS(toProcess)
	out.FilteredOut = len(toProcess) - len(filtered)
	out.SMSAfterFilter = len(filtered)

	if len(filtered) == 0 {
		out.PatternUsed = uc.aiService.GetUsedService()
		return out, nil
	}

	chunksDone := 0
	for chunkStart := 0; chunkStart < len(filtered) && chunksDone < maxAIChunks; chunkStart += smsPerChunk {
		chunkEnd := chunkStart + smsPerChunk
		if chunkEnd > len(filtered) {
			chunkEnd = len(filtered)
		}
		chunk := filtered[chunkStart:chunkEnd]
		chunksDone++
		out.ChunksProcessed++

		var sb strings.Builder
		lineToBody := make(map[int]string)
		lineNum := 0
		for _, msg := range chunk {
			body := sanitizeSMSLine(msg.Body, maxBodyRunes)
			if body == "" {
				continue
			}
			lineNum++
			lineToBody[lineNum] = msg.Body
			sb.WriteString(fmt.Sprintf("[%d] %s\n", lineNum, body))
		}
		block := sb.String()
		if strings.TrimSpace(block) == "" {
			continue
		}

		resp, err := uc.aiService.ExtractTransactionsFromSMSChunk(ctx, block)
		if err != nil || resp == nil {
			out.ProcessingErrors++
			continue
		}

		seenLine := make(map[int]bool)
		for _, line := range resp.Lines {
			if line.Line < 1 || seenLine[line.Line] {
				continue
			}
			seenLine[line.Line] = true

			raw, ok := lineToBody[line.Line]
			if !ok {
				out.ProcessingErrors++
				continue
			}

			ext := batchTxnLineToExtraction(line, raw)

			if !ext.Success {
				out.NotBankSms++
				continue
			}
			if ext.Confidence < minAutoConfidence {
				out.LowConfidenceOrSkipped++
				continue
			}
			if ext.TransactionType != "income" && ext.TransactionType != "expense" && ext.TransactionType != "transfer" {
				out.LowConfidenceOrSkipped++
				continue
			}
			if ext.Amount <= 0 || ext.Amount > 1e9 {
				out.LowConfidenceOrSkipped++
				continue
			}

			_, err := uc.createTransactionFromAIExtraction(userID, ext, raw)
			if err != nil {
				out.ProcessingErrors++
				continue
			}
			out.TransactionsCreated++
		}
	}

	out.PatternUsed = uc.aiService.GetUsedService()
	return out, nil
}

func batchTxnLineToExtraction(line webapi.BatchSMSTransactionLine, raw string) *webapi.TransactionExtraction {
	return &webapi.TransactionExtraction{
		Success:         line.Success,
		Amount:          line.Amount,
		Description:     line.Description,
		Merchant:        line.Merchant,
		Date:            line.Date,
		TransactionType: line.TransactionType,
		Confidence:      line.Confidence,
		Currency:        line.Currency,
		RawMessage:      raw,
	}
}

func normalizeSlugForAnalytics(key string) string {
	k := strings.ToLower(strings.TrimSpace(key))
	k = strings.ReplaceAll(k, "-", "_")
	if _, ok := budgetCategorySlugToName[k]; ok {
		return k
	}
	switch k {
	case "groceries", "restaurant", "dining":
		return "food"
	case "gas", "fuel", "mobility":
		return "transport"
	case "fun", "leisure":
		return "entertainment"
	case "bills", "services":
		return "utilities"
	case "medical", "pharmacy":
		return "health"
	case "retail":
		return "shopping"
	case "school", "learning":
		return "education"
	default:
		return "other"
	}
}

// slug de IA → nombre de categoría por defecto en Money Flow
var budgetCategorySlugToName = map[string]string{
	"food":          "Alimentación",
	"transport":     "Transporte",
	"entertainment": "Ocio",
	"utilities":     "Servicios",
	"health":        "Salud",
	"shopping":      "Compras",
	"education":     "Educación",
	"other":         "Otros",
}

func resolveBudgetCategory(categoryKey string, userCats []*entity.Category, fallback *entity.Category) *entity.Category {
	k := strings.ToLower(strings.TrimSpace(categoryKey))
	k = strings.ReplaceAll(k, "-", "_")
	if k == "" {
		return fallback
	}
	targetName, ok := budgetCategorySlugToName[k]
	if !ok {
		switch k {
		case "groceries", "restaurant", "dining":
			targetName = "Alimentación"
		case "gas", "fuel", "mobility":
			targetName = "Transporte"
		case "fun", "leisure":
			targetName = "Ocio"
		case "bills", "services":
			targetName = "Servicios"
		case "medical", "pharmacy":
			targetName = "Salud"
		case "retail":
			targetName = "Compras"
		case "school", "learning":
			targetName = "Educación"
		default:
			return fallback
		}
	}
	for _, c := range userCats {
		if strings.EqualFold(strings.TrimSpace(c.Name), targetName) {
			return c
		}
	}
	return fallback
}

// sortSMSMessagesNewestFirst ordena por fecha descendente (ISO8601 en Date); sin fecha van al final.
func sortSMSMessagesNewestFirst(messages []dto.SMSMessageForAnalysis) {
	sort.SliceStable(messages, func(i, j int) bool {
		ti := parseSMSAnalysisDate(messages[i].Date)
		tj := parseSMSAnalysisDate(messages[j].Date)
		if ti.IsZero() && tj.IsZero() {
			return false
		}
		if ti.IsZero() {
			return false
		}
		if tj.IsZero() {
			return true
		}
		return ti.After(tj)
	})
}

func parseSMSAnalysisDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func sanitizeSMSLine(body string, maxRunes int) string {
	s := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(body), "\n", " "), "\r", " ")
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxRunes])
}

// GetNotificationStats obtiene estadísticas de procesamiento de notificaciones
func (uc *BankNotificationPatternUseCase) GetNotificationStats(userID *uint, days int) (*dto.NotificationStatsResponse, error) {
	// TODO: Implementar estadísticas reales
	// Por ahora retornamos estadísticas de ejemplo
	stats := &dto.NotificationStatsResponse{
		TotalReceived:     0,
		TotalProcessed:    0,
		TotalFailed:       0,
		AutoCreated:       0,
		PendingValidation: 0,
		ByChannel:         make(map[string]int),
		ByBank:            make(map[string]int),
		ByDay:             []dto.DailyNotificationStat{},
		AverageConfidence: 0.0,
	}

	return stats, nil
}
