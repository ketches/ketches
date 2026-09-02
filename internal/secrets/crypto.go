package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/ketches/ketches/internal/app"
)

const (
	encryptedEnvelopePrefix = "enc:"
	encryptedV1Prefix       = encryptedEnvelopePrefix + "v1:"
	encryptedV2Prefix       = encryptedEnvelopePrefix + "v2:"
	keyIDLength             = 12
)

type encryptionKey struct {
	id  string
	key []byte
}

func EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := currentEncryptionKey()
	if err != nil {
		return "", err
	}
	payload, err := seal([]byte(plaintext), key.key)
	if err != nil {
		return "", err
	}
	return encryptedV1Prefix + key.id + ":" + base64.StdEncoding.EncodeToString(payload), nil
}

func DecryptString(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	keys, err := encryptionKeys()
	if err != nil {
		return "", err
	}

	switch {
	case strings.HasPrefix(value, encryptedV2Prefix):
		encoded := strings.TrimPrefix(value, encryptedV2Prefix)
		keyID, payloadText, ok := strings.Cut(encoded, ":")
		if !ok || keyID == "" || payloadText == "" {
			return "", app.NewErrorf("invalid v2 ciphertext envelope")
		}
		payload, err := base64.StdEncoding.DecodeString(payloadText)
		if err != nil {
			return "", app.WrapErrorf(err, "decode ciphertext: %w", err)
		}
		for _, key := range keys {
			if key.id != keyID {
				continue
			}
			plaintext, err := open(payload, key.key)
			if err != nil {
				return "", err
			}
			return string(plaintext), nil
		}
		return "", app.NewErrorf("decrypt ciphertext: key %q is not configured", keyID)
	case strings.HasPrefix(value, encryptedV1Prefix):
		encoded := strings.TrimPrefix(value, encryptedV1Prefix)
		if keyID, payloadText, ok := strings.Cut(encoded, ":"); ok && len(keyID) == keyIDLength {
			payload, err := base64.StdEncoding.DecodeString(payloadText)
			if err != nil {
				return "", app.WrapErrorf(err, "decode ciphertext: %w", err)
			}
			for _, key := range keys {
				if key.id != keyID {
					continue
				}
				plaintext, err := open(payload, key.key)
				if err != nil {
					return "", err
				}
				return string(plaintext), nil
			}
			return "", app.NewErrorf("decrypt ciphertext: key %q is not configured", keyID)
		}
		payload, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", app.WrapErrorf(err, "decode ciphertext: %w", err)
		}
		var decryptErr error
		for _, key := range keys {
			plaintext, err := open(payload, key.key)
			if err == nil {
				return string(plaintext), nil
			}
			decryptErr = err
		}
		return "", decryptErr
	default:
		return "", app.NewErrorf("ciphertext has an unsupported envelope")
	}
}

// DecryptStringCompatible reads legacy plaintext during the data-migration
// window. New writes must always use EncryptString.
func DecryptStringCompatible(value string) (string, error) {
	// Only values without an encryption envelope may be treated as legacy
	// plaintext. An unknown envelope must fail closed instead of being copied
	// into a workload as if it were plaintext.
	if !strings.HasPrefix(value, encryptedEnvelopePrefix) {
		return value, nil
	}
	return DecryptString(value)
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedEnvelopePrefix)
}

func NeedsRotation(value string) bool {
	if !strings.HasPrefix(value, encryptedV1Prefix) {
		return true
	}
	key, err := currentEncryptionKey()
	if err != nil {
		return true
	}
	return !strings.HasPrefix(value, encryptedV1Prefix+key.id+":")
}

func seal(plaintext, key []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, app.WrapErrorf(err, "read nonce: %w", err)
	}
	return append(nonce, gcm.Seal(nil, nonce, plaintext, nil)...), nil
}

func open(payload, key []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize() {
		return nil, app.NewErrorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil {
		return nil, app.WrapErrorf(err, "decrypt ciphertext: %w", err)
	}
	return plaintext, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, app.WrapErrorf(err, "create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, app.WrapErrorf(err, "create gcm: %w", err)
	}
	return gcm, nil
}

func currentEncryptionKey() (encryptionKey, error) {
	value := strings.TrimSpace(app.Config.SecretEncryptionKey)
	if value == "" {
		return encryptionKey{}, app.NewErrorf("secret encryption key is not configured")
	}
	return deriveEncryptionKey(value), nil
}

func encryptionKeys() ([]encryptionKey, error) {
	current, err := currentEncryptionKey()
	if err != nil {
		return nil, err
	}
	keys := []encryptionKey{current}
	seen := map[string]struct{}{current.id: {}}
	for _, value := range strings.Split(app.Config.PreviousSecretEncryptionKeys, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := deriveEncryptionKey(value)
		if _, ok := seen[key.id]; ok {
			continue
		}
		seen[key.id] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, errors.New("no secret encryption keys are configured")
	}
	return keys, nil
}

func deriveEncryptionKey(value string) encryptionKey {
	sum := sha256.Sum256([]byte(value))
	return encryptionKey{
		id:  hex.EncodeToString(sum[:])[:keyIDLength],
		key: sum[:],
	}
}
