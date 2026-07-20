package services

import (
	"testing"
)

func TestValidateGitRepositoryURLRejectsLocalAndUnsafeSchemes(t *testing.T) {
	for _, repoURL := range []string{
		"/var/lib/repos/project.git",
		"./project.git",
		"file:///var/lib/repos/project.git",
		"http://github.com/org/project.git",
		"https://127.0.0.1/project.git",
		"git@127.0.0.1:org/project.git",
		"ssh://metadata.google.internal/org/project.git",
	} {
		if err := validateGitRepositoryURL(repoURL); err == nil {
			t.Errorf("expected unsafe git URL %q to be rejected", repoURL)
		}
	}
}

func TestValidateGitRepositoryURLAcceptsPublicHTTPSAndSSHForms(t *testing.T) {
	for _, repoURL := range []string{
		"https://github.com/org/project.git",
		"ssh://git@github.com/org/project.git",
		"git@github.com:org/project.git",
	} {
		if err := validateGitRepositoryURL(repoURL); err != nil {
			t.Errorf("expected valid git URL %q: %v", repoURL, err)
		}
	}
}
