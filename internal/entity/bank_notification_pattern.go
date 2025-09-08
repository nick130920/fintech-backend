package entity

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// NotificationChannel define los canales de notificación
type NotificationChannel string

const (
	NotificationChannelSMS   NotificationChannel = "sms"   // Mensaje SMS
	NotificationChannelPush  NotificationChannel = "push"  // Notificación push
	NotificationChannelEmail NotificationChannel = "email" // Correo electrónico
	NotificationChannelApp   NotificationChannel = "app"   // Notificación interna de app bancaria
)

// NotificationPatternStatus define el estado del patrón
type NotificationPatternStatus string

const (
	NotificationPatternStatusActive   NotificationPatternStatus = "active"   // Activo
	NotificationPatternStatusInactive NotificationPatternStatus = "inactive" // Inactivo
	NotificationPatternStatusLearning NotificationPatternStatus = "learning" // En aprendizaje
)

// BankNotificationPattern representa un patrón de notificación bancaria
type BankNotificationPattern struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relaciones
	UserID        uint `json:"user_id" gorm:"not null;index"`
	BankAccountID uint `json:"bank_account_id" gorm:"not null;index"`

	// Información del patrón
	Name        string                    `json:"name" gorm:"not null" validate:"required,min=1,max=100"`
	Description string                    `json:"description" validate:"max=500"`
	Channel     NotificationChannel       `json:"channel" gorm:"not null" validate:"required,oneof=sms push email app"`
	Status      NotificationPatternStatus `json:"status" gorm:"default:'active'" validate:"oneof=active inactive learning"`

	// Patrón de mensaje
	MessagePattern  string `json:"message_pattern" gorm:"type:text" validate:"max=2000"` // Patrón regex o template
	ExampleMessage  string `json:"example_message" gorm:"type:text" validate:"max=2000"` // Ejemplo de mensaje
	KeywordsTrigger string `json:"keywords_trigger" validate:"max=1000"`                 // Palabras clave (JSON array)
	KeywordsExclude string `json:"keywords_exclude" validate:"max=1000"`                 // Palabras a excluir (JSON array)

	// Configuración de extracción
	AmountRegex      string `json:"amount_regex" validate:"max=500"`      // Regex para extraer monto
	DateRegex        string `json:"date_regex" validate:"max=500"`        // Regex para extraer fecha
	DescriptionRegex string `json:"description_regex" validate:"max=500"` // Regex para extraer descripción
	MerchantRegex    string `json:"merchant_regex" validate:"max=500"`    // Regex para extraer comercio

	// Configuración de validación
	RequiresValidation  bool    `json:"requires_validation" gorm:"default:true"`                   // Si requiere validación manual
	ConfidenceThreshold float64 `json:"confidence_threshold" gorm:"default:0.8;type:decimal(3,2)"` // Umbral de confianza (0-1)
	AutoApprove         bool    `json:"auto_approve" gorm:"default:false"`                         // Auto-aprobar si confianza > umbral

	// Estadísticas de uso
	MatchCount    int        `json:"match_count" gorm:"default:0"`                    // Número de coincidencias
	SuccessCount  int        `json:"success_count" gorm:"default:0"`                  // Número de éxitos
	SuccessRate   float64    `json:"success_rate" gorm:"default:0;type:decimal(5,2)"` // Tasa de éxito (%)
	LastMatchedAt *time.Time `json:"last_matched_at"`                                 // Última vez que coincidió

	// Configuración adicional
	Priority  int    `json:"priority" gorm:"default:100"`     // Prioridad del patrón (menor = mayor prioridad)
	IsDefault bool   `json:"is_default" gorm:"default:false"` // Si es el patrón por defecto para el banco
	Tags      string `json:"tags" validate:"max:500"`         // Tags adicionales (JSON array)
	Metadata  string `json:"metadata" gorm:"type:text"`       // Metadatos adicionales (JSON)
}

// GetKeywordsTrigger convierte el campo KeywordsTrigger (JSON string) a slice de strings
func (bnp *BankNotificationPattern) GetKeywordsTrigger() []string {
	if bnp.KeywordsTrigger == "" {
		return []string{}
	}

	var keywords []string
	if err := json.Unmarshal([]byte(bnp.KeywordsTrigger), &keywords); err != nil {
		return []string{}
	}

	return keywords
}

// SetKeywordsTrigger convierte un slice de strings a JSON string para KeywordsTrigger
func (bnp *BankNotificationPattern) SetKeywordsTrigger(keywords []string) error {
	if len(keywords) == 0 {
		bnp.KeywordsTrigger = ""
		return nil
	}

	jsonKeywords, err := json.Marshal(keywords)
	if err != nil {
		return err
	}

	bnp.KeywordsTrigger = string(jsonKeywords)
	return nil
}

// GetKeywordsExclude convierte el campo KeywordsExclude (JSON string) a slice de strings
func (bnp *BankNotificationPattern) GetKeywordsExclude() []string {
	if bnp.KeywordsExclude == "" {
		return []string{}
	}

	var keywords []string
	if err := json.Unmarshal([]byte(bnp.KeywordsExclude), &keywords); err != nil {
		return []string{}
	}

	return keywords
}

// SetKeywordsExclude convierte un slice de strings a JSON string para KeywordsExclude
func (bnp *BankNotificationPattern) SetKeywordsExclude(keywords []string) error {
	if len(keywords) == 0 {
		bnp.KeywordsExclude = ""
		return nil
	}

	jsonKeywords, err := json.Marshal(keywords)
	if err != nil {
		return err
	}

	bnp.KeywordsExclude = string(jsonKeywords)
	return nil
}

// GetTags convierte el campo Tags (JSON string) a slice de strings
func (bnp *BankNotificationPattern) GetTags() []string {
	if bnp.Tags == "" {
		return []string{}
	}

	var tags []string
	if err := json.Unmarshal([]byte(bnp.Tags), &tags); err != nil {
		return []string{}
	}

	return tags
}

// SetTags convierte un slice de strings a JSON string para el campo Tags
func (bnp *BankNotificationPattern) SetTags(tags []string) error {
	if len(tags) == 0 {
		bnp.Tags = ""
		return nil
	}

	jsonTags, err := json.Marshal(tags)
	if err != nil {
		return err
	}

	bnp.Tags = string(jsonTags)
	return nil
}

// GetMetadata convierte el campo Metadata (JSON string) a map
func (bnp *BankNotificationPattern) GetMetadata() map[string]interface{} {
	if bnp.Metadata == "" {
		return make(map[string]interface{})
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(bnp.Metadata), &metadata); err != nil {
		return make(map[string]interface{})
	}

	return metadata
}

// SetMetadata convierte un map a JSON string para el campo Metadata
func (bnp *BankNotificationPattern) SetMetadata(metadata map[string]interface{}) error {
	if len(metadata) == 0 {
		bnp.Metadata = ""
		return nil
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	bnp.Metadata = string(jsonMetadata)
	return nil
}

// IsActive verifica si el patrón está activo
func (bnp *BankNotificationPattern) IsActive() bool {
	return bnp.Status == NotificationPatternStatusActive
}

// IsLearning verifica si el patrón está en modo aprendizaje
func (bnp *BankNotificationPattern) IsLearning() bool {
	return bnp.Status == NotificationPatternStatusLearning
}

// CanAutoApprove verifica si puede auto-aprobar basado en la confianza
func (bnp *BankNotificationPattern) CanAutoApprove(confidence float64) bool {
	return bnp.AutoApprove && confidence >= bnp.ConfidenceThreshold
}

// RecordMatch registra una coincidencia del patrón
func (bnp *BankNotificationPattern) RecordMatch(success bool) {
	bnp.MatchCount++
	if success {
		bnp.SuccessCount++
	}

	// Calcular tasa de éxito
	if bnp.MatchCount > 0 {
		bnp.SuccessRate = float64(bnp.SuccessCount) / float64(bnp.MatchCount) * 100
	}

	now := time.Now()
	bnp.LastMatchedAt = &now
}

// GetDisplayName retorna el nombre de visualización del patrón
func (bnp *BankNotificationPattern) GetDisplayName() string {
	if bnp.Name != "" {
		return bnp.Name
	}
	return string(bnp.Channel) + " Pattern"
}

// BankNotificationPatternSummary representa un resumen del patrón
type BankNotificationPatternSummary struct {
	ID                  uint                      `json:"id"`
	Name                string                    `json:"name"`
	Channel             NotificationChannel       `json:"channel"`
	Status              NotificationPatternStatus `json:"status"`
	MatchCount          int                       `json:"match_count"`
	SuccessRate         float64                   `json:"success_rate"`
	ConfidenceThreshold float64                   `json:"confidence_threshold"`
	AutoApprove         bool                      `json:"auto_approve"`
	IsDefault           bool                      `json:"is_default"`
	LastMatchedAt       *time.Time                `json:"last_matched_at"`
	BankAccountAlias    string                    `json:"bank_account_alias"`
	BankName            string                    `json:"bank_name"`
}

// ToSummary convierte un BankNotificationPattern a BankNotificationPatternSummary
func (bnp *BankNotificationPattern) ToSummary() BankNotificationPatternSummary {
	return BankNotificationPatternSummary{
		ID:                  bnp.ID,
		Name:                bnp.GetDisplayName(),
		Channel:             bnp.Channel,
		Status:              bnp.Status,
		MatchCount:          bnp.MatchCount,
		SuccessRate:         bnp.SuccessRate,
		ConfidenceThreshold: bnp.ConfidenceThreshold,
		AutoApprove:         bnp.AutoApprove,
		IsDefault:           bnp.IsDefault,
		LastMatchedAt:       bnp.LastMatchedAt,
	}
}

// BankNotificationPatternFilter representa filtros para búsqueda de patrones
type BankNotificationPatternFilter struct {
	BankAccountID *uint                      `json:"bank_account_id"`
	Channel       *NotificationChannel       `json:"channel"`
	Status        *NotificationPatternStatus `json:"status"`
	IsDefault     *bool                      `json:"is_default"`
	Search        string                     `json:"search"` // Búsqueda en nombre o descripción
	Limit         int                        `json:"limit"`
	Offset        int                        `json:"offset"`
	OrderBy       string                     `json:"order_by"`  // Campo por el cual ordenar
	OrderDir      string                     `json:"order_dir"` // ASC o DESC
}
