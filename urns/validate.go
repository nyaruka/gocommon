package urns

import (
	"gopkg.in/go-playground/validator.v9"
)

// RegisterValidation registers the URN validate tags
func RegisterValidation(validate *validator.Validate) {
	validate.RegisterValidation("urnscheme", ValidateURNScheme)
}

// ValidateURNScheme validates whether the field valus is a valid URN scheme
func ValidateURNScheme(fl validator.FieldLevel) bool {
	_, valid := validSchemes[fl.Field().String()]
	return valid
}
