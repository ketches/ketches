package api

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// ValidateSlug checks if a string is a valid slug.
// A valid slug must be 3-32 characters long, start with a letter,
// end with a letter or number, and contain only lowercase letters, numbers, and hyphens.
func ValidateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	if len(slug) < 3 || len(slug) > 32 {
		return false
	}
	// Must start with a letter, not end with a hyphen, and contain only lowercase letters, numbers, and hyphens.
	re := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !re.MatchString(slug) {
		return false
	}
	// A more specific check to ensure it doesn't end with a hyphen,
	// as the regex above already handles most cases.
	if slug[len(slug)-1] == '-' {
		return false
	}
	return true
}
