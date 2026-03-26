package v1

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"github.com/nick130920/fintech-backend/pkg/currency"
	"github.com/nick130920/fintech-backend/pkg/logger"
)

// ValidationErrorDetail representa un error de validación específico
type ValidationErrorDetail struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
}

// CustomValidator wrapper para validator con mensajes personalizados
type CustomValidator struct {
	validator *validator.Validate
	messages  map[string]string
}

// NewCustomValidator crea un nuevo validador personalizado
func NewCustomValidator() *CustomValidator {
	v := validator.New()

	// Registrar validaciones personalizadas
	registerCustomValidations(v)

	cv := &CustomValidator{
		validator: v,
		messages:  getCustomMessages(),
	}

	// Usar nombres de campo del JSON tag
	cv.validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return cv
}

// Validate valida una estructura y devuelve errores detallados
func (cv *CustomValidator) Validate(s interface{}) error {
	err := cv.validator.Struct(s)
	if err == nil {
		return nil
	}

	var validationErrors []ValidationErrorDetail

	for _, err := range err.(validator.ValidationErrors) {
		detail := ValidationErrorDetail{
			Field:   err.Field(),
			Value:   err.Value(),
			Tag:     err.Tag(),
			Message: cv.getErrorMessage(err),
		}
		validationErrors = append(validationErrors, detail)
	}

	// Log de errores de validación
	log := logger.Get()
	log.Debug().
		Interface("validation_errors", validationErrors).
		Str("struct_type", reflect.TypeOf(s).Name()).
		Msg("🔍 VALIDATION ERRORS")

	return apperrors.ErrValidation.
		WithDetails("Los datos proporcionados no son válidos").
		WithField("validation_errors", validationErrors)
}

// ValidationMiddleware middleware para manejar errores de validación automáticamente
func ValidationMiddleware(cv *CustomValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("validator", cv)
		c.Next()
	}
}

// BindAndValidate helper para bind y validación automática
func BindAndValidate(c *gin.Context, obj interface{}) error {
	// Bind del JSON
	if err := c.ShouldBindJSON(obj); err != nil {
		log := logger.Get()
		log.Debug().
			Err(err).
			Str("path", c.Request.URL.Path).
			Msg("🚫 JSON BIND ERROR")

		return apperrors.ErrInvalidRequest.
			WithDetails("Formato JSON inválido").
			WithField("bind_error", err.Error())
	}

	// Validación
	validator, exists := c.Get("validator")
	if !exists {
		return apperrors.ErrInternal.WithDetails("Validador no configurado")
	}

	cv := validator.(*CustomValidator)
	return cv.Validate(obj)
}

// getErrorMessage obtiene mensaje personalizado para un error de validación
func (cv *CustomValidator) getErrorMessage(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()

	// Buscar mensaje específico para campo+tag
	if msg, exists := cv.messages[field+"."+tag]; exists {
		return fmt.Sprintf(msg, fe.Value())
	}

	// Buscar mensaje genérico para tag
	if msg, exists := cv.messages[tag]; exists {
		return fmt.Sprintf(msg, field, fe.Value())
	}

	// Mensaje por defecto
	return fmt.Sprintf("El campo '%s' no cumple con la validación '%s'", field, tag)
}

// registerCustomValidations registra validaciones personalizadas
func registerCustomValidations(v *validator.Validate) {
	v.RegisterValidation("currency", func(fl validator.FieldLevel) bool {
		return currency.IsValid(fl.Field().String())
	})

	// Validación para tipo de cuenta bancaria
	v.RegisterValidation("bank_account_type", func(fl validator.FieldLevel) bool {
		accountType := fl.Field().String()
		validTypes := []string{"checking", "savings", "credit", "debit", "investment"}

		for _, valid := range validTypes {
			if accountType == valid {
				return true
			}
		}
		return false
	})

	// Validación para color hexadecimal
	v.RegisterValidation("hexcolor", func(fl validator.FieldLevel) bool {
		color := fl.Field().String()
		if len(color) != 7 || color[0] != '#' {
			return false
		}

		for i := 1; i < len(color); i++ {
			c := color[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	})

	// Validación para número de teléfono internacional
	v.RegisterValidation("international_phone", func(fl validator.FieldLevel) bool {
		phone := fl.Field().String()
		// Formato básico: +[código país][número]
		if len(phone) < 8 || phone[0] != '+' {
			return false
		}

		for i := 1; i < len(phone); i++ {
			c := phone[i]
			if !(c >= '0' && c <= '9') && c != ' ' && c != '-' {
				return false
			}
		}
		return true
	})

	// Validación para contraseña segura
	v.RegisterValidation("secure_password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		if len(password) < 8 {
			return false
		}

		hasUpper := false
		hasLower := false
		hasDigit := false
		hasSpecial := false

		for _, c := range password {
			switch {
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= '0' && c <= '9':
				hasDigit = true
			default:
				hasSpecial = true
			}
		}

		return hasUpper && hasLower && hasDigit && hasSpecial
	})
}

// getCustomMessages devuelve mensajes personalizados para validaciones
func getCustomMessages() map[string]string {
	return map[string]string{
		// Mensajes genéricos
		"required": "El campo '%s' es obligatorio",
		"min":      "El campo '%s' debe tener al menos %v caracteres",
		"max":      "El campo '%s' no puede tener más de %v caracteres",
		"email":    "El campo '%s' debe ser un email válido",
		"gte":      "El campo '%s' debe ser mayor o igual a %v",
		"gt":       "El campo '%s' debe ser mayor que %v",
		"lte":      "El campo '%s' debe ser menor o igual a %v",
		"lt":       "El campo '%s' debe ser menor que %v",
		"oneof":    "El campo '%s' debe ser uno de los valores permitidos",

		// Mensajes personalizados
		"currency":            "El campo '%s' debe ser un código de moneda ISO 4217 válido",
		"bank_account_type":   "El campo '%s' debe ser un tipo de cuenta válido (checking, savings, credit, debit, investment)",
		"hexcolor":            "El campo '%s' debe ser un color hexadecimal válido (ej: #FF0000)",
		"international_phone": "El campo '%s' debe ser un número de teléfono internacional válido (ej: +52 55 1234 5678)",
		"secure_password":     "El campo '%s' debe tener al menos 8 caracteres con mayúsculas, minúsculas, números y símbolos",

		// Mensajes específicos para campos conocidos
		"bank_name.required":           "El nombre del banco es obligatorio",
		"bank_name.min":                "El nombre del banco debe tener al menos 2 caracteres",
		"bank_name.max":                "El nombre del banco no puede tener más de 100 caracteres",
		"account_alias.required":       "El alias de la cuenta es obligatorio",
		"account_number_mask.required": "Los últimos dígitos de la cuenta son obligatorios",
		"amount.required":              "El monto es obligatorio",
		"amount.gt":                    "El monto debe ser mayor que 0",
		"description.required":         "La descripción es obligatoria",
		"category_id.required":         "La categoría es obligatoria",
		"email.email":                  "El formato del email no es válido",
		"password.secure_password":     "La contraseña debe tener al menos 8 caracteres con mayúsculas, minúsculas, números y símbolos",
	}
}

// ValidateAndRespond helper que valida y responde automáticamente en caso de error
func ValidateAndRespond(c *gin.Context, obj interface{}) bool {
	if err := BindAndValidate(c, obj); err != nil {
		AbortWithError(c, err)
		return false
	}
	return true
}
