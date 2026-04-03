package services

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

var ErrInvalidGitRepositoryURL = errors.New("git repository URL must use http, https, ssh, or git@")

func validateGitRepositoryURL(repoURL string) error {
	trimmed := strings.TrimSpace(repoURL)
	if trimmed == "" {
		return ErrInvalidGitRepositoryURL
	}
	if strings.HasPrefix(trimmed, "git@") {
		return nil
	}
	if filepath.IsAbs(trimmed) || strings.HasPrefix(trimmed, "./") || strings.HasPrefix(trimmed, "../") {
		if strings.HasPrefix(trimmed, "-") {
			return ErrInvalidGitRepositoryURL
		}
		return nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ErrInvalidGitRepositoryURL
	}
	switch parsed.Scheme {
	case "http", "https", "ssh":
		if strings.TrimSpace(parsed.Host) == "" {
			return ErrInvalidGitRepositoryURL
		}
		return nil
	default:
		return ErrInvalidGitRepositoryURL
	}
}
