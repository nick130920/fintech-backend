package apperrors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_BuildersAndTypeChecks(t *testing.T) {
	base := NewAppError(ErrCodeValidation, "fallo validacion", http.StatusBadRequest).
		WithDetails("campo amount inválido").
		WithField("amount", -1)

	internalErr := errors.New("db timeout")
	base.WithInternal(internalErr)

	if base.Details == "" || base.Fields["amount"] != -1 {
		t.Fatalf("expected details/fields to be populated")
	}
	if _, ok := IsAppError(base); !ok {
		t.Fatalf("expected IsAppError to detect AppError")
	}
	if base.Error() == "" {
		t.Fatalf("expected non-empty Error() string")
	}
}

func TestWrapError(t *testing.T) {
	original := errors.New("raw error")
	wrapped := WrapError(original, ErrCodeInternal, "wrapped", http.StatusInternalServerError)

	if wrapped.Code != ErrCodeInternal || wrapped.Message != "wrapped" {
		t.Fatalf("unexpected wrapped metadata: %+v", wrapped)
	}
	if wrapped.Internal == nil || wrapped.Internal.Error() != "raw error" {
		t.Fatalf("expected wrapped internal error")
	}
}
