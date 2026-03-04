package validator

import (
	"regexp"

	"go-standard/internal/apperror"

	"github.com/go-playground/validator/v10"
)

var indonesianPhoneRegex = regexp.MustCompile(`^(\+62|62|0)[0-9]{8,13}$`)

// New creates a shared *validator.Validate instance with custom rules registered.
func New() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("indonesian_phone", validateIndonesianPhone)
	return v
}

// validateIndonesianPhone checks that a field matches the Indonesian phone number format.
func validateIndonesianPhone(fl validator.FieldLevel) bool {
	return indonesianPhoneRegex.MatchString(fl.Field().String())
}

// ValidateStruct validates any struct and maps validation errors into an AppError.
// Returns nil when validation passes.
func ValidateStruct(v *validator.Validate, s interface{}) *apperror.AppError {
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	var details []apperror.FieldError
	for _, fe := range err.(validator.ValidationErrors) {
		details = append(details, apperror.FieldError{
			Field:   fe.Field(),
			Message: buildMessage(fe),
		})
	}

	return apperror.BadRequestWithDetails("validation failed", details)
}

// buildMessage constructs a human-readable message for a single validation failure.
func buildMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "invalid email format"
	case "min":
		return "value is too short or too small (min: " + fe.Param() + ")"
	case "max":
		return "value is too long or too large (max: " + fe.Param() + ")"
	case "indonesian_phone":
		return "invalid Indonesian phone number format"
	default:
		return "invalid value (" + fe.Tag() + ")"
	}
}
