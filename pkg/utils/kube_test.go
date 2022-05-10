package utils

import (
	"fmt"
	"testing"
)

func TestValidateResourceName(t *testing.T) {
	fmt.Println(ValidateResourceName(""))
	fmt.Println(ValidateResourceName("e"))
	fmt.Println(ValidateResourceName("2"))
	fmt.Println(ValidateResourceName("e-23"))
	fmt.Println(ValidateResourceName("e.23"))
	fmt.Println(ValidateResourceName("E23"))
	fmt.Println(ValidateResourceName("E-23"))
	fmt.Println(ValidateResourceName("E.23"))
	fmt.Println(ValidateResourceName("e,23"))
	fmt.Println(ValidateResourceName("e汉23"))
	fmt.Println(ValidateResourceName("e，23"))
	fmt.Println(ValidateResourceName("e@23"))
}
