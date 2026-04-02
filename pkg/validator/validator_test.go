package validator

import "testing"

type sampleDTO struct {
	Email string `validate:"required,email"`
	Color string `validate:"hexcolor"`
}

func TestValidator_ValidateAndValidateStruct(t *testing.T) {
	v := New()

	err := v.Validate(sampleDTO{Email: "not-an-email", Color: "#12FG"})
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	fieldErrors := v.ValidateStruct(sampleDTO{Email: "bad", Color: "123456"})
	if _, ok := fieldErrors["email"]; !ok {
		t.Fatalf("expected email field error")
	}
	if _, ok := fieldErrors["color"]; !ok {
		t.Fatalf("expected color field error")
	}
}

func TestValidator_ValidatePassword(t *testing.T) {
	v := New()

	if err := v.ValidatePassword("weakpass"); err == nil {
		t.Fatalf("expected error for password without uppercase/number")
	}
	if err := v.ValidatePassword("StrongPass1"); err != nil {
		t.Fatalf("expected strong password to pass, got %v", err)
	}
}

func TestValidator_IsValidCurrency(t *testing.T) {
	v := New()
	if !v.IsValidCurrency("usd") {
		t.Fatalf("expected USD (lowercase input) to be valid")
	}
	if v.IsValidCurrency("ZZZ") {
		t.Fatalf("expected ZZZ to be invalid")
	}
}
