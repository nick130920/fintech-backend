package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator envuelve el validador de go-playground
type Validator struct {
	validate *validator.Validate
}

// New crea una nueva instancia del validador
func New() *Validator {
	validate := validator.New()

	// Registrar validaciones personalizadas
	validate.RegisterValidation("hexcolor", validateHexColor)

	return &Validator{
		validate: validate,
	}
}

// Validate valida una estructura usando las etiquetas de validación
func (v *Validator) Validate(s interface{}) error {
	if err := v.validate.Struct(s); err != nil {
		// Convertir errores de validación a un formato más amigable
		var validationErrors []string

		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, v.formatValidationError(err))
		}

		return errors.New(strings.Join(validationErrors, "; "))
	}
	return nil
}

// formatValidationError formatea un error de validación a un mensaje legible
func (v *Validator) formatValidationError(err validator.FieldError) string {
	field := strings.ToLower(err.Field())

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s es requerido", field)
	case "email":
		return fmt.Sprintf("%s debe ser un email válido", field)
	case "min":
		return fmt.Sprintf("%s debe tener al menos %s caracteres", field, err.Param())
	case "max":
		return fmt.Sprintf("%s no puede tener más de %s caracteres", field, err.Param())
	case "len":
		return fmt.Sprintf("%s debe tener exactamente %s caracteres", field, err.Param())
	case "gt":
		return fmt.Sprintf("%s debe ser mayor que %s", field, err.Param())
	case "gte":
		return fmt.Sprintf("%s debe ser mayor o igual que %s", field, err.Param())
	case "lt":
		return fmt.Sprintf("%s debe ser menor que %s", field, err.Param())
	case "lte":
		return fmt.Sprintf("%s debe ser menor o igual que %s", field, err.Param())
	case "oneof":
		return fmt.Sprintf("%s debe ser uno de: %s", field, err.Param())
	case "hexcolor":
		return fmt.Sprintf("%s debe ser un color hexadecimal válido", field)
	default:
		return fmt.Sprintf("%s no cumple con la validación %s", field, err.Tag())
	}
}

// ValidateVar valida una variable individual
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// RegisterValidation registra una función de validación personalizada
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// validateHexColor valida que una cadena sea un color hexadecimal válido
func validateHexColor(fl validator.FieldLevel) bool {
	color := fl.Field().String()

	// Permitir valores vacíos (se manejan con omitempty)
	if color == "" {
		return true
	}

	// Debe empezar con #
	if !strings.HasPrefix(color, "#") {
		return false
	}

	// Debe tener 4 o 7 caracteres (#RGB o #RRGGBB)
	if len(color) != 4 && len(color) != 7 {
		return false
	}

	// Verificar que todos los caracteres después de # sean hexadecimales
	for _, char := range color[1:] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}

	return true
}

// ValidateEmail valida específicamente un email
func (v *Validator) ValidateEmail(email string) error {
	return v.ValidateVar(email, "required,email")
}

// ValidatePassword valida específicamente una contraseña
func (v *Validator) ValidatePassword(password string) error {
	if err := v.ValidateVar(password, "required,min=8"); err != nil {
		return err
	}

	// Validaciones adicionales de contraseña
	hasUpper := false
	hasLower := false
	hasNumber := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		}
	}

	if !hasUpper {
		return errors.New("la contraseña debe contener al menos una letra mayúscula")
	}
	if !hasLower {
		return errors.New("la contraseña debe contener al menos una letra minúscula")
	}
	if !hasNumber {
		return errors.New("la contraseña debe contener al menos un número")
	}

	return nil
}

// ValidateStruct valida una estructura y retorna errores mapeados por campo
func (v *Validator) ValidateStruct(s interface{}) map[string]string {
	errors := make(map[string]string)

	if err := v.validate.Struct(s); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			field := strings.ToLower(err.Field())
			errors[field] = v.formatValidationError(err)
		}
	}

	return errors
}

// IsValidCurrency verifica si una moneda es válida (código ISO 4217)
func (v *Validator) IsValidCurrency(currency string) bool {
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true,
		"MXN": true, "CAD": true, "AUD": true, "CHF": true,
		"CNY": true, "BRL": true, "ARS": true, "COP": true,
	}

	return validCurrencies[strings.ToUpper(currency)]
}
