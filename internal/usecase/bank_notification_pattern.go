package usecase

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// BankNotificationPatternUseCase contiene la lógica de negocio para patrones de notificación bancaria
type BankNotificationPatternUseCase struct {
	patternRepo     repo.BankNotificationPatternRepo
	bankAccountRepo repo.BankAccountRepo
	userRepo        repo.UserRepo
}

// NewBankNotificationPatternUseCase crea una nueva instancia de BankNotificationPatternUseCase
func NewBankNotificationPatternUseCase(
	patternRepo repo.BankNotificationPatternRepo,
	bankAccountRepo repo.BankAccountRepo,
	userRepo repo.UserRepo,
) *BankNotificationPatternUseCase {
	return &BankNotificationPatternUseCase{
		patternRepo:     patternRepo,
		bankAccountRepo: bankAccountRepo,
		userRepo:        userRepo,
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

// ProcessNotification procesa una notificación bancaria usando patrones
func (uc *BankNotificationPatternUseCase) ProcessNotification(userID, bankAccountID uint, channel entity.NotificationChannel, message string) (*dto.ProcessedNotificationResponse, error) {
	// Verificar que la cuenta bancaria pertenece al usuario
	bankAccount, err := uc.bankAccountRepo.GetByID(bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil || bankAccount.UserID != userID {
		return nil, errors.New("unauthorized access to bank account")
	}

	// Obtener patrones que podrían coincidir
	matchingPatterns, err := uc.patternRepo.GetMatchingPatterns(bankAccountID, channel, message)
	if err != nil {
		return nil, fmt.Errorf("failed to get matching patterns: %w", err)
	}

	// Si no hay patrones coincidentes, usar el patrón por defecto
	if len(matchingPatterns) == 0 {
		defaultPattern, err := uc.patternRepo.GetDefaultPattern(bankAccountID, channel)
		if err != nil {
			return nil, fmt.Errorf("failed to get default pattern: %w", err)
		}
		if defaultPattern != nil {
			matchingPatterns = []*entity.BankNotificationPattern{defaultPattern}
		}
	}

	// Procesar con el primer patrón coincidente (mayor prioridad)
	var bestPattern *entity.BankNotificationPattern
	var confidence float64
	var extractedData map[string]interface{}

	if len(matchingPatterns) > 0 {
		bestPattern = matchingPatterns[0]
		extractedData, confidence = uc.extractDataFromMessage(bestPattern, message)
	}

	response := &dto.ProcessedNotificationResponse{
		BankAccountID: bankAccountID,
		Channel:       channel,
		Message:       message,
		Processed:     bestPattern != nil,
		Confidence:    confidence,
		ExtractedData: extractedData,
	}

	if bestPattern != nil {
		response.PatternID = &bestPattern.ID
		response.PatternName = bestPattern.Name
		response.RequiresValidation = bestPattern.RequiresValidation || confidence < bestPattern.ConfidenceThreshold
	}

	return response, nil
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
