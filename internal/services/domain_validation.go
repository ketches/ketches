package services

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

func normalizeDomainValue(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}

	if strings.HasPrefix(normalized, "*.") {
		if errs := validation.IsWildcardDNS1123Subdomain(normalized); len(errs) > 0 {
			return "", fmt.Errorf("domain %q is invalid: %s", value, strings.Join(errs, ", "))
		}
		return normalized, nil
	}

	if errs := validation.IsDNS1123Subdomain(normalized); len(errs) > 0 {
		return "", fmt.Errorf("domain %q is invalid: %s", value, strings.Join(errs, ", "))
	}

	return normalized, nil
}
