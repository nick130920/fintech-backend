package repo

import "github.com/nick130920/fintech-backend/internal/entity"

// BankNotificationPatternRepo define la interfaz para operaciones de patrones de notificación bancaria
type BankNotificationPatternRepo interface {
	// Operaciones básicas CRUD
	Create(pattern *entity.BankNotificationPattern) error
	GetByID(id uint) (*entity.BankNotificationPattern, error)
	Update(pattern *entity.BankNotificationPattern) error
	Delete(id uint) error

	// Operaciones específicas del usuario y cuenta bancaria
	GetByUserID(userID uint) ([]*entity.BankNotificationPattern, error)
	GetByBankAccountID(bankAccountID uint) ([]*entity.BankNotificationPattern, error)
	GetActiveByBankAccountID(bankAccountID uint) ([]*entity.BankNotificationPattern, error)

	// Operaciones de búsqueda y filtros
	GetByChannel(userID uint, channel entity.NotificationChannel) ([]*entity.BankNotificationPattern, error)
	GetByStatus(userID uint, status entity.NotificationPatternStatus) ([]*entity.BankNotificationPattern, error)
	GetWithFilters(userID uint, filter entity.BankNotificationPatternFilter) ([]*entity.BankNotificationPattern, int64, error)

	// Operaciones de coincidencia y procesamiento
	GetMatchingPatterns(bankAccountID uint, channel entity.NotificationChannel, message string) ([]*entity.BankNotificationPattern, error)
	GetDefaultPattern(bankAccountID uint, channel entity.NotificationChannel) (*entity.BankNotificationPattern, error)
	GetByPriority(bankAccountID uint, channel entity.NotificationChannel) ([]*entity.BankNotificationPattern, error)

	// Operaciones de estado y configuración
	SetStatus(id uint, status entity.NotificationPatternStatus) error
	SetDefault(id uint, isDefault bool) error
	UpdateStatistics(id uint, matchCount, successCount int, successRate float64) error

	// Estadísticas y consultas especiales
	CountByUserID(userID uint) (int64, error)
	CountByBankAccountID(bankAccountID uint) (int64, error)
	GetSummaryByUserID(userID uint) ([]*entity.BankNotificationPatternSummary, error)
	GetTopPerformingPatterns(userID uint, limit int) ([]*entity.BankNotificationPattern, error)
	GetLearningPatterns(userID uint) ([]*entity.BankNotificationPattern, error)
}
