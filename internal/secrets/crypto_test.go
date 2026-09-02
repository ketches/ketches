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

	if _, err := DecryptString("plaintext"); err == nil {
		t.Fatalf("DecryptString() error = nil, want non-nil")
	}
}

func TestDecryptStringCompatibleRejectsUnsupportedEnvelope(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	app.Config.SecretEncryptionKey = "test-master-key"
	for _, value := range []string{"enc:v3:opaque", "enc:malformed"} {
		t.Run(value, func(t *testing.T) {
			if plaintext, err := DecryptStringCompatible(value); err == nil {
				t.Fatalf("DecryptStringCompatible() = %q, error = nil; want an error", plaintext)
			}
		})
	}
}

func TestDecryptStringCompatibleAllowsLegacyPlaintext(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	app.Config.SecretEncryptionKey = "test-master-key"
	plaintext, err := DecryptStringCompatible("legacy-plaintext")
	if err != nil {
		t.Fatalf("DecryptStringCompatible() error = %v", err)
	}
	if plaintext != "legacy-plaintext" {
		t.Fatalf("DecryptStringCompatible() = %q, want legacy plaintext", plaintext)
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

func TestDecryptStringUsesPreviousKeyDuringRotation(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })

	app.Config.SecretEncryptionKey = "old-master-key"
	ciphertext, err := EncryptString("rotated-secret")
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}

	app.Config.SecretEncryptionKey = "new-master-key"
	app.Config.PreviousSecretEncryptionKeys = "old-master-key"
	plaintext, err := DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("DecryptString() error = %v", err)
	}
	if plaintext != "rotated-secret" {
		t.Fatalf("DecryptString() = %q", plaintext)
	}
	if !NeedsRotation(ciphertext) {
		t.Fatal("old ciphertext should require rotation")
	}
}
