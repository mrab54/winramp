package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"runtime"
)

// Encryption provides methods for encrypting sensitive configuration values
type Encryption struct {
	key []byte
}

// NewEncryption creates a new encryption instance using a machine-specific key
func NewEncryption() (*Encryption, error) {
	key, err := getMachineKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get machine key: %w", err)
	}
	
	return &Encryption{
		key: key,
	}, nil
}

// getMachineKey derives a machine-specific encryption key
func getMachineKey() ([]byte, error) {
	var seed string
	
	if runtime.GOOS == "windows" {
		// Use Windows machine GUID
		seed = os.Getenv("COMPUTERNAME")
		if seed == "" {
			seed = "default-windows-key"
		}
		// Add user SID for additional entropy
		userSID := os.Getenv("USERDOMAIN_ROAMINGPROFILE")
		if userSID != "" {
			seed += userSID
		}
	} else {
		// For non-Windows systems, use hostname
		hostname, err := os.Hostname()
		if err != nil {
			seed = "default-key"
		} else {
			seed = hostname
		}
	}
	
	// Add application-specific salt
	seed += "WinRamp-Config-Salt-2024"
	
	// Generate 32-byte key using SHA256
	hash := sha256.Sum256([]byte(seed))
	return hash[:], nil
}

// Encrypt encrypts a string value
func (e *Encryption) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to create nonce: %w", err)
	}
	
	// Encrypt and seal
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a string value
func (e *Encryption) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	
	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return string(plaintext), nil
}

// EncryptField encrypts a specific configuration field
func (e *Encryption) EncryptField(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Only encrypt if it looks like a sensitive value
		if isSensitive(v) {
			encrypted, err := e.Encrypt(v)
			if err == nil {
				return "encrypted:" + encrypted
			}
		}
		return v
	default:
		return value
	}
}

// DecryptField decrypts a specific configuration field
func (e *Encryption) DecryptField(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		if len(v) > 10 && v[:10] == "encrypted:" {
			decrypted, err := e.Decrypt(v[10:])
			if err == nil {
				return decrypted
			}
		}
		return v
	default:
		return value
	}
}

// isSensitive checks if a value appears to be sensitive
func isSensitive(value string) bool {
	// Check for common sensitive patterns
	if len(value) < 8 {
		return false
	}
	
	// Look for password-like strings or tokens
	hasUpperCase := false
	hasLowerCase := false
	hasDigit := false
	hasSpecial := false
	
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpperCase = true
		case r >= 'a' && r <= 'z':
			hasLowerCase = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '!' || r == '@' || r == '#' || r == '$' || r == '%' || r == '^' || r == '&' || r == '*':
			hasSpecial = true
		}
	}
	
	// Consider it sensitive if it has mixed case and digits or special chars
	return (hasUpperCase && hasLowerCase) && (hasDigit || hasSpecial)
}