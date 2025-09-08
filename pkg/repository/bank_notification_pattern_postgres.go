package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"gorm.io/gorm"
)

// BankNotificationPatternPostgres implementa BankNotificationPatternRepo usando PostgreSQL
type BankNotificationPatternPostgres struct {
	db *gorm.DB
}

// NewBankNotificationPatternPostgres crea una nueva instancia del repositorio de patrones de notificación
func NewBankNotificationPatternPostgres(db *gorm.DB) repo.BankNotificationPatternRepo {
	return &BankNotificationPatternPostgres{db: db}
}

// Create crea un nuevo patrón de notificación bancaria
func (r *BankNotificationPatternPostgres) Create(pattern *entity.BankNotificationPattern) error {
	if err := r.db.Create(pattern).Error; err != nil {
		return fmt.Errorf("failed to create bank notification pattern: %w", err)
	}
	return nil
}

// GetByID obtiene un patrón de notificación por ID
func (r *BankNotificationPatternPostgres) GetByID(id uint) (*entity.BankNotificationPattern, error) {
	var pattern entity.BankNotificationPattern
	if err := r.db.First(&pattern, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bank notification pattern by id %d: %w", id, err)
	}
	return &pattern, nil
}

// Update actualiza un patrón de notificación existente
func (r *BankNotificationPatternPostgres) Update(pattern *entity.BankNotificationPattern) error {
	if err := r.db.Save(pattern).Error; err != nil {
		return fmt.Errorf("failed to update bank notification pattern: %w", err)
	}
	return nil
}

// Delete elimina un patrón de notificación (soft delete)
func (r *BankNotificationPatternPostgres) Delete(id uint) error {
	if err := r.db.Delete(&entity.BankNotificationPattern{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete bank notification pattern: %w", err)
	}
	return nil
}

// GetByUserID obtiene todos los patrones de notificación de un usuario
func (r *BankNotificationPatternPostgres) GetByUserID(userID uint) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("user_id = ?", userID).Order("priority ASC, created_at DESC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank notification patterns for user %d: %w", userID, err)
	}
	return patterns, nil
}

// GetByBankAccountID obtiene todos los patrones de notificación de una cuenta bancaria
func (r *BankNotificationPatternPostgres) GetByBankAccountID(bankAccountID uint) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("bank_account_id = ?", bankAccountID).Order("priority ASC, created_at DESC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank notification patterns for account %d: %w", bankAccountID, err)
	}
	return patterns, nil
}

// GetActiveByBankAccountID obtiene patrones activos de una cuenta bancaria
func (r *BankNotificationPatternPostgres) GetActiveByBankAccountID(bankAccountID uint) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("bank_account_id = ? AND status = ?", bankAccountID, entity.NotificationPatternStatusActive).
		Order("priority ASC, created_at DESC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get active bank notification patterns for account %d: %w", bankAccountID, err)
	}
	return patterns, nil
}

// GetByChannel obtiene patrones de notificación por canal
func (r *BankNotificationPatternPostgres) GetByChannel(userID uint, channel entity.NotificationChannel) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("user_id = ? AND channel = ?", userID, channel).Order("priority ASC, created_at DESC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank notification patterns by channel for user %d: %w", userID, err)
	}
	return patterns, nil
}

// GetByStatus obtiene patrones de notificación por estado
func (r *BankNotificationPatternPostgres) GetByStatus(userID uint, status entity.NotificationPatternStatus) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("user_id = ? AND status = ?", userID, status).Order("priority ASC, created_at DESC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank notification patterns by status for user %d: %w", userID, err)
	}
	return patterns, nil
}

// GetWithFilters obtiene patrones con filtros
func (r *BankNotificationPatternPostgres) GetWithFilters(userID uint, filter entity.BankNotificationPatternFilter) ([]*entity.BankNotificationPattern, int64, error) {
	query := r.db.Where("user_id = ?", userID)

	// Aplicar filtros
	if filter.BankAccountID != nil {
		query = query.Where("bank_account_id = ?", *filter.BankAccountID)
	}
	if filter.Channel != nil {
		query = query.Where("channel = ?", *filter.Channel)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.Search != "" {
		query = query.Where("(name ILIKE ? OR description ILIKE ?)", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// Contar total
	var total int64
	if err := query.Model(&entity.BankNotificationPattern{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count bank notification patterns: %w", err)
	}

	// Aplicar ordenamiento
	orderBy := "priority"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	orderDir := "ASC"
	if filter.OrderDir != "" {
		orderDir = filter.OrderDir
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, orderDir))

	// Aplicar paginación
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var patterns []*entity.BankNotificationPattern
	if err := query.Find(&patterns).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get bank notification patterns with filters: %w", err)
	}

	return patterns, total, nil
}

// GetMatchingPatterns obtiene patrones que podrían coincidir con un mensaje
func (r *BankNotificationPatternPostgres) GetMatchingPatterns(bankAccountID uint, channel entity.NotificationChannel, message string) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern

	// Obtener patrones activos para la cuenta y canal
	if err := r.db.Where("bank_account_id = ? AND channel = ? AND status = ?",
		bankAccountID, channel, entity.NotificationPatternStatusActive).
		Order("priority ASC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get matching patterns: %w", err)
	}

	// Filtrar patrones que podrían coincidir basándose en palabras clave
	var matchingPatterns []*entity.BankNotificationPattern
	messageLower := strings.ToLower(message)

	for _, pattern := range patterns {
		// Verificar palabras clave de activación
		triggerKeywords := pattern.GetKeywordsTrigger()
		excludeKeywords := pattern.GetKeywordsExclude()

		hasMatch := len(triggerKeywords) == 0 // Si no hay keywords, considera todos
		hasExclusion := false

		// Verificar palabras de activación
		for _, keyword := range triggerKeywords {
			if strings.Contains(messageLower, strings.ToLower(keyword)) {
				hasMatch = true
				break
			}
		}

		// Verificar palabras de exclusión
		for _, keyword := range excludeKeywords {
			if strings.Contains(messageLower, strings.ToLower(keyword)) {
				hasExclusion = true
				break
			}
		}

		if hasMatch && !hasExclusion {
			matchingPatterns = append(matchingPatterns, pattern)
		}
	}

	return matchingPatterns, nil
}

// GetDefaultPattern obtiene el patrón por defecto para una cuenta y canal
func (r *BankNotificationPatternPostgres) GetDefaultPattern(bankAccountID uint, channel entity.NotificationChannel) (*entity.BankNotificationPattern, error) {
	var pattern entity.BankNotificationPattern
	if err := r.db.Where("bank_account_id = ? AND channel = ? AND is_default = ? AND status = ?",
		bankAccountID, channel, true, entity.NotificationPatternStatusActive).First(&pattern).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get default pattern: %w", err)
	}
	return &pattern, nil
}

// GetByPriority obtiene patrones ordenados por prioridad
func (r *BankNotificationPatternPostgres) GetByPriority(bankAccountID uint, channel entity.NotificationChannel) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("bank_account_id = ? AND channel = ? AND status = ?",
		bankAccountID, channel, entity.NotificationPatternStatusActive).
		Order("priority ASC, success_rate DESC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get patterns by priority: %w", err)
	}
	return patterns, nil
}

// SetStatus cambia el estado de un patrón
func (r *BankNotificationPatternPostgres) SetStatus(id uint, status entity.NotificationPatternStatus) error {
	if err := r.db.Model(&entity.BankNotificationPattern{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to set status for pattern %d: %w", id, err)
	}
	return nil
}

// SetDefault establece un patrón como por defecto
func (r *BankNotificationPatternPostgres) SetDefault(id uint, isDefault bool) error {
	// Si se está estableciendo como por defecto, desactivar otros patrones por defecto
	if isDefault {
		var pattern entity.BankNotificationPattern
		if err := r.db.First(&pattern, id).Error; err != nil {
			return fmt.Errorf("failed to get pattern for default setting: %w", err)
		}

		// Desactivar otros patrones por defecto para la misma cuenta y canal
		if err := r.db.Model(&entity.BankNotificationPattern{}).
			Where("bank_account_id = ? AND channel = ? AND id != ?", pattern.BankAccountID, pattern.Channel, id).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to unset other default patterns: %w", err)
		}
	}

	if err := r.db.Model(&entity.BankNotificationPattern{}).Where("id = ?", id).Update("is_default", isDefault).Error; err != nil {
		return fmt.Errorf("failed to set default for pattern %d: %w", id, err)
	}
	return nil
}

// UpdateStatistics actualiza las estadísticas de un patrón
func (r *BankNotificationPatternPostgres) UpdateStatistics(id uint, matchCount, successCount int, successRate float64) error {
	updates := map[string]interface{}{
		"match_count":     matchCount,
		"success_count":   successCount,
		"success_rate":    successRate,
		"last_matched_at": time.Now(),
	}
	if err := r.db.Model(&entity.BankNotificationPattern{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update statistics for pattern %d: %w", id, err)
	}
	return nil
}

// CountByUserID cuenta los patrones de un usuario
func (r *BankNotificationPatternPostgres) CountByUserID(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.BankNotificationPattern{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count patterns for user %d: %w", userID, err)
	}
	return count, nil
}

// CountByBankAccountID cuenta los patrones de una cuenta bancaria
func (r *BankNotificationPatternPostgres) CountByBankAccountID(bankAccountID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.BankNotificationPattern{}).Where("bank_account_id = ?", bankAccountID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count patterns for bank account %d: %w", bankAccountID, err)
	}
	return count, nil
}

// GetSummaryByUserID obtiene un resumen de patrones de un usuario
func (r *BankNotificationPatternPostgres) GetSummaryByUserID(userID uint) ([]*entity.BankNotificationPatternSummary, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Preload("BankAccount").Where("user_id = ?", userID).Order("priority ASC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get patterns for summary: %w", err)
	}

	summaries := make([]*entity.BankNotificationPatternSummary, len(patterns))
	for i, pattern := range patterns {
		summary := pattern.ToSummary()
		summaries[i] = &summary
	}

	return summaries, nil
}

// GetTopPerformingPatterns obtiene los patrones con mejor rendimiento
func (r *BankNotificationPatternPostgres) GetTopPerformingPatterns(userID uint, limit int) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("user_id = ? AND match_count > 0", userID).
		Order("success_rate DESC, match_count DESC").Limit(limit).Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get top performing patterns for user %d: %w", userID, err)
	}
	return patterns, nil
}

// GetLearningPatterns obtiene patrones en modo aprendizaje
func (r *BankNotificationPatternPostgres) GetLearningPatterns(userID uint) ([]*entity.BankNotificationPattern, error) {
	var patterns []*entity.BankNotificationPattern
	if err := r.db.Where("user_id = ? AND status = ?", userID, entity.NotificationPatternStatusLearning).
		Order("created_at DESC").Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get learning patterns for user %d: %w", userID, err)
	}
	return patterns, nil
}
