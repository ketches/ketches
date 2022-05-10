package utils

import (
	"errors"
	"fmt"
	"unicode"
)

func ValidateResourceName(raw string) error {
	if len(raw) == 0 {
		return errors.New("the resource name length must be greater than 0")
	}

	for _, c := range raw {
		if !unicode.IsLower(c) && !unicode.IsDigit(c) && c != '.' && c != '-' {
			return errors.New(fmt.Sprintf("the resource name [%s] can only contain lowercase characters, digit characters, '-' and '.'", raw))
		}
	}
	return nil
}
