package envelope

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type keyFile struct {
	Version   int    `json:"version"`
	Scheme    string `json:"scheme"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	KeyB64    string `json:"key_b64"`
}

// GenerateKey creates a fresh simulator/development AES-256-GCM key.
func GenerateKey() (Key, error) {
	keyBytes := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
		return Key{}, fmt.Errorf("generate key bytes: %w", err)
	}
	keyIDRandom := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, keyIDRandom); err != nil {
		return Key{}, fmt.Errorf("generate key id: %w", err)
	}
	return Key{
		Version:   keyVersion,
		Scheme:    SchemeV1,
		Algorithm: AlgorithmAES256GCM,
		KeyID:     "kid_" + base64.RawURLEncoding.EncodeToString(keyIDRandom),
		Key:       keyBytes,
	}, nil
}

// LoadKeyFile loads the simulator-only JSON key file format.
func LoadKeyFile(path string) (Key, error) {
	if path == "" {
		return Key{}, fmt.Errorf("key file path is required")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return Key{}, err
	}
	var file keyFile
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return Key{}, fmt.Errorf("decode key file: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return Key{}, fmt.Errorf("decode key file: %w", err)
	}
	keyBytes, err := base64.RawURLEncoding.DecodeString(file.KeyB64)
	if err != nil {
		return Key{}, fmt.Errorf("decode key_b64: %w", err)
	}
	key := Key{
		Version:   file.Version,
		Scheme:    file.Scheme,
		Algorithm: file.Algorithm,
		KeyID:     file.KeyID,
		Key:       keyBytes,
	}
	if err := validateKey(key); err != nil {
		return Key{}, err
	}
	return key, nil
}

// SaveKeyFile writes the simulator-only JSON key file format with restrictive
// permissions where supported by the host filesystem.
func SaveKeyFile(path string, key Key) error {
	if path == "" {
		return fmt.Errorf("key file path is required")
	}
	if err := validateKey(key); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create key file directory: %w", err)
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	writeErr := encoder.Encode(keyFile{
		Version:   key.Version,
		Scheme:    key.Scheme,
		Algorithm: key.Algorithm,
		KeyID:     key.KeyID,
		KeyB64:    base64.RawURLEncoding.EncodeToString(key.Key),
	})
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write key file: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close key file: %w", closeErr)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("set key file permissions: %w", err)
	}
	return nil
}

func validateKey(key Key) error {
	if key.Version != keyVersion {
		return fmt.Errorf("unsupported key version %d", key.Version)
	}
	if key.Scheme != SchemeV1 {
		return fmt.Errorf("unsupported key scheme %q", key.Scheme)
	}
	if key.Algorithm != AlgorithmAES256GCM {
		return fmt.Errorf("unsupported key algorithm %q", key.Algorithm)
	}
	if key.KeyID == "" {
		return fmt.Errorf("key_id is required")
	}
	if len(key.Key) != keySize {
		return fmt.Errorf("key must be %d bytes", keySize)
	}
	return nil
}
