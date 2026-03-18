package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
	patternRepo     repo.BankNotificationPatternRepo
	bankAccountRepo repo.BankAccountRepo
	userRepo        repo.UserRepo
	transactionRepo repo.TransactionRepo
	expenseRepo     repo.ExpenseRepo
	incomeRepo      repo.IncomeRepo
	budgetRepo      repo.BudgetRepo
	categoryRepo    repo.CategoryRepo
	aiService       *webapi.AIServiceWithFallback
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
	aiService *webapi.AIServiceWithFallback,
) *BankNotificationPatternUseCase {
	return &BankNotificationPatternUseCase{
		patternRepo:     patternRepo,
		bankAccountRepo: bankAccountRepo,
		userRepo:        userRepo,
		transactionRepo: transactionRepo,
		expenseRepo:     expenseRepo,
		incomeRepo:      incomeRepo,
		budgetRepo:      budgetRepo,
		categoryRepo:    categoryRepo,
		aiService:       aiService,
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
			income.Currency = "MXN"
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
		expense.Currency = "MXN"
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
func (uc *BankNotificationPatternUseCase) ProcessSMSBatchForSuggestions(ctx context.Context, userID uint, messages []dto.SMSMessageForAnalysis) (*dto.AnalyzeSMSBatchResponse, error) {
	const (
		maxMessages   = 500
		smsPerChunk   = 28
		maxBodyRunes  = 220
		minConfidence = 0.3
	)
	if len(messages) > maxMessages {
		messages = messages[:maxMessages]
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

	filtered := filterMessagesLikelyBankSMS(messages)
	if len(filtered) < 8 {
		filtered = filterNonEmptyBodies(messages, maxMessages)
	}

	var totalExpense float64
	var expenseCount int

	for chunkStart := 0; chunkStart < len(filtered); chunkStart += smsPerChunk {
		chunkEnd := chunkStart + smsPerChunk
		if chunkEnd > len(filtered) {
			chunkEnd = len(filtered)
		}
		chunk := filtered[chunkStart:chunkEnd]
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
			totalExpense += line.Amount
			expenseCount++
		}
	}

	byCategory := []dto.BudgetSuggestionCategory{}
	if totalExpense > 0 && expenseCount > 0 {
		byCategory = append(byCategory, dto.BudgetSuggestionCategory{
			CategoryID:   otrosCategory.ID,
			CategoryName: otrosCategory.Name,
			Total:        totalExpense,
			Count:        expenseCount,
		})
	}

	return &dto.AnalyzeSMSBatchResponse{
		Suggestions: dto.BudgetSuggestions{
			TotalExpense3m: totalExpense,
			ByCategory:     byCategory,
		},
	}, nil
}

// bankTransactionSMSRegexp: una pasada sobre el texto; agrupa señales típicas de SMS bancarios LATAM.
// Monedas / símbolos · verbos de movimiento · instituciones · identificadores de pago regionales.
var bankTransactionSMSRegexp = regexp.MustCompile(
	`(?i)(` +
		`[\$€£]|` +
		`\b(mxn|cop|clp|ars|brl|pen|usd|eur|gs\.|pyg|uyu|bob|crc|gtq|hnl|nio|dop|ves)\b|` +
		`compra|consumo|pago|débito|debito|debit|cargo|abono|retiro|transferencia|transfer|` +
		`\bbanco\b|` +
		`spei|clabe|cbu|cvu|alias|pix|` +
		`bbva|banamex|santander|banorte|hsbc|scotiabank|inbursa|azteca|banregio|` +
		`bancolombia|nequi|daviplata|davivienda|banco\s+de\s+occidente|` +
		`interbank|bcp|continental|nubank|mercado.pago|rappi\s*bank|uala|brubank|` +
		`yape|plin|banco\s+nacion|banco\s+estado|` +
		`tarjeta|cuenta|ahorro|corriente|movimiento|transacci|` +
		`visa|mastercard|amex` +
		`)`,
)

func filterMessagesLikelyBankSMS(messages []dto.SMSMessageForAnalysis) []dto.SMSMessageForAnalysis {
	var out []dto.SMSMessageForAnalysis
	for _, msg := range messages {
		if likelyBankTransactionSMS(msg.Body) {
			out = append(out, msg)
		}
	}
	return out
}

func filterNonEmptyBodies(messages []dto.SMSMessageForAnalysis, max int) []dto.SMSMessageForAnalysis {
	var out []dto.SMSMessageForAnalysis
	for _, msg := range messages {
		if strings.TrimSpace(msg.Body) != "" {
			out = append(out, msg)
			if len(out) >= max {
				break
			}
		}
	}
	return out
}

func likelyBankTransactionSMS(body string) bool {
	b := strings.TrimSpace(body)
	if utf8.RuneCountInString(b) < 12 {
		return false
	}
	if !strings.ContainsAny(b, "0123456789") {
		return false
	}
	if bankTransactionSMSRegexp.MatchString(b) {
		return true
	}
	// Mensajes largos con dígitos: posible extracto o notificación sin palabras clave estándar
	return utf8.RuneCountInString(b) > 80
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
