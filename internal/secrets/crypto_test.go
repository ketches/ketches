package secrets

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
)

func TestEncryptDecryptString(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	app.Config.SecretEncryptionKey = "test-master-key"

	ciphertext, err := EncryptString("kubeconfig-data")
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}
	if ciphertext == "kubeconfig-data" {
		t.Fatalf("EncryptString() returned plaintext")
	}

	plaintext, err := DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("DecryptString() error = %v", err)
	}
	if plaintext != "kubeconfig-data" {
		t.Fatalf("DecryptString() = %q, want %q", plaintext, "kubeconfig-data")
	}
}

func TestDecryptStringRejectsPlaintextWithoutPrefix(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	app.Config.SecretEncryptionKey = "test-master-key"

	if _, err := DecryptString("legacy-plaintext"); err == nil {
		t.Fatalf("DecryptString() error = nil, want non-nil")
	}
}

func TestEncryptStringFailsWithoutConfiguredKey(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	app.Config.SecretEncryptionKey = ""

	if _, err := EncryptString("kubeconfig-data"); err == nil {
		t.Fatalf("EncryptString() error = nil, want non-nil")
	}
}
