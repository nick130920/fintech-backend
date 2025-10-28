package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"github.com/nick130920/fintech-backend/pkg/webapi"
)

// BankNotificationPatternUseCase contiene la lógica de negocio para patrones de notificación bancaria
type BankNotificationPatternUseCase struct {
	patternRepo     repo.BankNotificationPatternRepo
	bankAccountRepo repo.BankAccountRepo
	userRepo        repo.UserRepo
	transactionRepo repo.TransactionRepo
	geminiService   *webapi.GeminiService
}

// NewBankNotificationPatternUseCase crea una nueva instancia de BankNotificationPatternUseCase
func NewBankNotificationPatternUseCase(
	patternRepo repo.BankNotificationPatternRepo,
	bankAccountRepo repo.BankAccountRepo,
	userRepo repo.UserRepo,
	transactionRepo repo.TransactionRepo,
	geminiService *webapi.GeminiService,
) *BankNotificationPatternUseCase {
	return &BankNotificationPatternUseCase{
		patternRepo:     patternRepo,
		bankAccountRepo: bankAccountRepo,
		userRepo:        userRepo,
		transactionRepo: transactionRepo,
		geminiService:   geminiService,
	}
}

// CreatePattern crea un nuevo patrón de notificación bancaria
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

	// Validar regex si se proporcionan
	if err := uc.validateRegexPatterns(req); err != nil {
		return nil, fmt.Errorf("invalid regex patterns: %w", err)
	}

	// Crear la entidad patrón
	pattern := &entity.BankNotificationPattern{
		UserID:              userID,
		BankAccountID:       req.BankAccountID,
		Name:                req.Name,
		Description:         req.Description,
		Channel:             req.Channel,
		Status:              entity.NotificationPatternStatusActive,
		MessagePattern:      req.MessagePattern,
		ExampleMessage:      req.ExampleMessage,
		AmountRegex:         req.AmountRegex,
		DateRegex:           req.DateRegex,
		DescriptionRegex:    req.DescriptionRegex,
		MerchantRegex:       req.MerchantRegex,
		RequiresValidation:  req.RequiresValidation,
		ConfidenceThreshold: req.ConfidenceThreshold,
		AutoApprove:         req.AutoApprove,
		Priority:            req.Priority,
		IsDefault:           req.IsDefault,
	}

	// Establecer palabras clave
	if len(req.KeywordsTrigger) > 0 {
		if err := pattern.SetKeywordsTrigger(req.KeywordsTrigger); err != nil {
			return nil, fmt.Errorf("failed to set trigger keywords: %w", err)
		}
	}
	if len(req.KeywordsExclude) > 0 {
		if err := pattern.SetKeywordsExclude(req.KeywordsExclude); err != nil {
			return nil, fmt.Errorf("failed to set exclude keywords: %w", err)
		}
	}
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

	// Si se está marcando como por defecto, desactivar otros patrones por defecto
	if req.IsDefault {
		if err := uc.unsetOtherDefaultPatterns(req.BankAccountID, req.Channel); err != nil {
			return nil, fmt.Errorf("failed to unset other default patterns: %w", err)
		}
	}

	// Crear el patrón
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

// UpdatePattern actualiza un patrón existente
func (uc *BankNotificationPatternUseCase) UpdatePattern(userID, patternID uint, req *dto.UpdateBankNotificationPatternRequest) (*dto.BankNotificationPatternResponse, error) {
	// Obtener el patrón existente
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
	if req.MessagePattern != nil && *req.MessagePattern != "" {
		pattern.MessagePattern = *req.MessagePattern
	}
	if req.ExampleMessage != nil && *req.ExampleMessage != "" {
		pattern.ExampleMessage = *req.ExampleMessage
	}
	if req.AmountRegex != nil && *req.AmountRegex != "" {
		// Validar regex
		if _, err := regexp.Compile(*req.AmountRegex); err != nil {
			return nil, fmt.Errorf("invalid amount regex: %w", err)
		}
		pattern.AmountRegex = *req.AmountRegex
	}
	if req.DateRegex != nil && *req.DateRegex != "" {
		if _, err := regexp.Compile(*req.DateRegex); err != nil {
			return nil, fmt.Errorf("invalid date regex: %w", err)
		}
		pattern.DateRegex = *req.DateRegex
	}
	if req.DescriptionRegex != nil && *req.DescriptionRegex != "" {
		if _, err := regexp.Compile(*req.DescriptionRegex); err != nil {
			return nil, fmt.Errorf("invalid description regex: %w", err)
		}
		pattern.DescriptionRegex = *req.DescriptionRegex
	}
	if req.MerchantRegex != nil && *req.MerchantRegex != "" {
		if _, err := regexp.Compile(*req.MerchantRegex); err != nil {
			return nil, fmt.Errorf("invalid merchant regex: %w", err)
		}
		pattern.MerchantRegex = *req.MerchantRegex
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
			// Desactivar otros patrones por defecto
			if err := uc.unsetOtherDefaultPatterns(pattern.BankAccountID, pattern.Channel); err != nil {
				return nil, fmt.Errorf("failed to unset other default patterns: %w", err)
			}
		}
		pattern.IsDefault = *req.IsDefault
	}

	// Actualizar palabras clave si se proporcionan
	if req.KeywordsTrigger != nil {
		if err := pattern.SetKeywordsTrigger(req.KeywordsTrigger); err != nil {
			return nil, fmt.Errorf("failed to set trigger keywords: %w", err)
		}
	}
	if req.KeywordsExclude != nil {
		if err := pattern.SetKeywordsExclude(req.KeywordsExclude); err != nil {
			return nil, fmt.Errorf("failed to set exclude keywords: %w", err)
		}
	}
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

// validateRegexPatterns valida los patrones regex
func (uc *BankNotificationPatternUseCase) validateRegexPatterns(req *dto.CreateBankNotificationPatternRequest) error {
	if req.AmountRegex != "" {
		if _, err := regexp.Compile(req.AmountRegex); err != nil {
			return fmt.Errorf("invalid amount regex: %w", err)
		}
	}
	if req.DateRegex != "" {
		if _, err := regexp.Compile(req.DateRegex); err != nil {
			return fmt.Errorf("invalid date regex: %w", err)
		}
	}
	if req.DescriptionRegex != "" {
		if _, err := regexp.Compile(req.DescriptionRegex); err != nil {
			return fmt.Errorf("invalid description regex: %w", err)
		}
	}
	if req.MerchantRegex != "" {
		if _, err := regexp.Compile(req.MerchantRegex); err != nil {
			return fmt.Errorf("invalid merchant regex: %w", err)
		}
	}
	return nil
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

// extractDataFromMessage extrae datos de un mensaje usando un patrón
func (uc *BankNotificationPatternUseCase) extractDataFromMessage(pattern *entity.BankNotificationPattern, message string) (map[string]interface{}, float64) {
	extractedData := make(map[string]interface{})
	confidence := 0.0
	matches := 0
	totalAttempts := 0

	// Extraer monto
	if pattern.AmountRegex != "" {
		totalAttempts++
		if re, err := regexp.Compile(pattern.AmountRegex); err == nil {
			if match := re.FindStringSubmatch(message); len(match) > 1 {
				extractedData["amount"] = strings.TrimSpace(match[1])
				matches++
			}
		}
	}

	// Extraer fecha
	if pattern.DateRegex != "" {
		totalAttempts++
		if re, err := regexp.Compile(pattern.DateRegex); err == nil {
			if match := re.FindStringSubmatch(message); len(match) > 1 {
				extractedData["date"] = strings.TrimSpace(match[1])
				matches++
			}
		}
	}

	// Extraer descripción
	if pattern.DescriptionRegex != "" {
		totalAttempts++
		if re, err := regexp.Compile(pattern.DescriptionRegex); err == nil {
			if match := re.FindStringSubmatch(message); len(match) > 1 {
				extractedData["description"] = strings.TrimSpace(match[1])
				matches++
			}
		}
	}

	// Extraer comercio
	if pattern.MerchantRegex != "" {
		totalAttempts++
		if re, err := regexp.Compile(pattern.MerchantRegex); err == nil {
			if match := re.FindStringSubmatch(message); len(match) > 1 {
				extractedData["merchant"] = strings.TrimSpace(match[1])
				matches++
			}
		}
	}

	// Calcular confianza basada en coincidencias
	if totalAttempts > 0 {
		confidence = float64(matches) / float64(totalAttempts)
	} else {
		confidence = 0.5 // Confianza base si no hay regex definidos
	}

	return extractedData, confidence
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
		MessagePattern:      pattern.MessagePattern,
		ExampleMessage:      pattern.ExampleMessage,
		KeywordsTrigger:     pattern.GetKeywordsTrigger(),
		KeywordsExclude:     pattern.GetKeywordsExclude(),
		AmountRegex:         pattern.AmountRegex,
		DateRegex:           pattern.DateRegex,
		DescriptionRegex:    pattern.DescriptionRegex,
		MerchantRegex:       pattern.MerchantRegex,
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

// ProcessNotificationWebhook procesa una notificación bancaria desde webhook
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

	// Obtener patrones activos del usuario
	patterns, err := uc.patternRepo.GetByUserID(userID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err).WithDetails("Error al obtener patrones de notificación")
	}

	// Filtrar patrones por canal
	var activePatterns []*entity.BankNotificationPattern
	for _, pattern := range patterns {
		if pattern.Status == "active" && (string(pattern.Channel) == req.Channel || string(pattern.Channel) == "all") {
			activePatterns = append(activePatterns, pattern)
		}
	}

	if len(activePatterns) == 0 {
		return &dto.ProcessedNotificationResponse{
			Success:            false,
			TransactionCreated: false,
			Confidence:         0.0,
			Reason:             "No hay patrones activos para este canal",
		}, nil
	}

	// Intentar procesar con cada patrón (ordenado por prioridad)
	var bestMatch *entity.BankNotificationPattern
	var bestData map[string]interface{}
	var bestConfidence float64

	for _, pattern := range activePatterns {
		data, confidence := uc.extractDataFromMessage(pattern, req.Message)

		if confidence > bestConfidence {
			bestMatch = pattern
			bestData = data
			bestConfidence = confidence
		}
	}

	response := &dto.ProcessedNotificationResponse{
		Success:    bestMatch != nil,
		Confidence: bestConfidence,
	}

	if bestMatch == nil {
		response.Reason = "Ningún patrón coincide con la notificación"
		return response, nil
	}

	// Actualizar estadísticas del patrón
	bestMatch.MatchCount++
	bestMatch.LastMatchedAt = &time.Time{}
	*bestMatch.LastMatchedAt = time.Now()

	// Verificar si se debe crear transacción automáticamente
	shouldAutoCreate := bestConfidence >= bestMatch.ConfidenceThreshold && bestMatch.AutoApprove

	if shouldAutoCreate {
		// Crear transacción automáticamente
		transaction, err := uc.createTransactionFromNotification(userID, bestMatch, bestData, req)
		if err != nil {
			// Marcar como requiere validación si falla la creación automática
			response.TransactionCreated = false
			response.RequiresValidation = true
			response.Reason = "Error al crear transacción automáticamente: " + err.Error()
		} else {
			response.TransactionCreated = true
			response.TransactionID = transaction.ID
			response.Amount = transaction.Amount
			response.Description = transaction.Description

			// Actualizar estadísticas de éxito
			bestMatch.SuccessCount++
			bestMatch.SuccessRate = float64(bestMatch.SuccessCount) / float64(bestMatch.MatchCount)
		}
	} else {
		response.RequiresValidation = true
		response.Reason = "Requiere validación manual (confianza insuficiente o patrón no auto-aprobado)"
	}

	// Guardar estadísticas actualizadas del patrón
	if err := uc.patternRepo.Update(bestMatch); err != nil {
		// Log error pero no fallar la respuesta
		// TODO: Log error
	}

	response.PatternUsed = bestMatch.Name
	return response, nil
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
