package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/open-proofline/server/internal/envelope"
)

const (
	wrappedKeyArtifactVersion  = 1
	wrappedKeyArtifactScope    = "simulator-development"
	wrappingAlgorithmAgeX25519 = "age-v1-x25519"
	wrappedKeyRecipientType    = "trusted_contact"

	contactKeyFileVersion      = 1
	defaultContactKeyFileName  = "proofline-sim-contact.key.json"
	defaultWrappedKeyContactID = "contact_dev_default"
	simulatorMediaKeySize      = 32
)

type contactKeyFile struct {
	Version           int       `json:"version"`
	Scope             string    `json:"scope"`
	ContactID         string    `json:"contact_id"`
	ContactKeyID      string    `json:"contact_key_id"`
	WrappingAlgorithm string    `json:"wrapping_algorithm"`
	Recipient         string    `json:"recipient"`
	Identity          string    `json:"identity"`
	CreatedAt         time.Time `json:"created_at"`
}

type wrappedKeyArtifact struct {
	Version     int                `json:"version"`
	Scope       string             `json:"scope"`
	IncidentID  string             `json:"incident_id"`
	StreamID    string             `json:"stream_id"`
	MediaKeyID  string             `json:"media_key_id"`
	CreatedAt   time.Time          `json:"created_at"`
	WrappedKeys []wrappedKeyRecord `json:"wrapped_keys"`
}

type wrappedKeyRecord struct {
	WrappedKeyID      string `json:"wrapped_key_id"`
	RecipientType     string `json:"recipient_type"`
	ContactID         string `json:"contact_id"`
	ContactKeyID      string `json:"contact_key_id"`
	WrappingAlgorithm string `json:"wrapping_algorithm"`
	WrappedKeyB64     string `json:"wrapped_key_b64"`
}

func prepareContactWrappedKey(out io.Writer, cfg config, incidentID, streamID string, mediaKey envelope.Key) (envelope.Key, bool, error) {
	if strings.TrimSpace(cfg.wrappedKeyOutput) == "" {
		return envelope.Key{}, false, nil
	}

	contact, err := loadOrCreateContactKeyFile(cfg.contactKeyFile, cfg.wrappedKeyContactID)
	if err != nil {
		return envelope.Key{}, false, err
	}
	artifact, created, err := loadOrCreateWrappedKeyArtifact(cfg.wrappedKeyOutput, contact, incidentID, streamID, mediaKey)
	if err != nil {
		return envelope.Key{}, false, err
	}
	unwrapped, err := unwrapWrappedMediaKey(artifact, contact)
	if err != nil {
		return envelope.Key{}, false, err
	}
	if unwrapped.KeyID != mediaKey.KeyID || !bytes.Equal(unwrapped.Key, mediaKey.Key) {
		return envelope.Key{}, false, fmt.Errorf("wrapped-key artifact does not match simulator media key")
	}

	if created {
		fmt.Fprintln(out, "Wrapped-key artifact written; path omitted from output.")
	} else {
		fmt.Fprintln(out, "Wrapped-key artifact loaded; path omitted from output.")
	}
	fmt.Fprintln(out, "Contact key file configured; path omitted from output.")
	fmt.Fprintln(out, "Verified contact unwrap for wrapped-key artifact.")
	fmt.Fprintln(out)
	return unwrapped, true, nil
}

func loadOrCreateContactKeyFile(path, contactID string) (contactKeyFile, error) {
	if strings.TrimSpace(path) == "" {
		return contactKeyFile{}, fmt.Errorf("contact key file path is required")
	}
	contact, err := loadContactKeyFile(path)
	if err == nil {
		if contact.ContactID != contactID {
			return contactKeyFile{}, fmt.Errorf("contact key file contact_id does not match requested contact")
		}
		return contact, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return contactKeyFile{}, safePathError("load contact key file", err)
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return contactKeyFile{}, fmt.Errorf("generate contact identity: %w", err)
	}
	recipient := identity.Recipient().String()
	contact = contactKeyFile{
		Version:           contactKeyFileVersion,
		Scope:             wrappedKeyArtifactScope,
		ContactID:         contactID,
		ContactKeyID:      contactKeyIDForRecipient(recipient),
		WrappingAlgorithm: wrappingAlgorithmAgeX25519,
		Recipient:         recipient,
		Identity:          identity.String(),
		CreatedAt:         time.Now().UTC(),
	}
	if err := saveContactKeyFile(path, contact); err != nil {
		return contactKeyFile{}, err
	}
	return contact, nil
}

func loadContactKeyFile(path string) (contactKeyFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return contactKeyFile{}, err
	}
	var contact contactKeyFile
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contact); err != nil {
		return contactKeyFile{}, fmt.Errorf("decode contact key file: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return contactKeyFile{}, fmt.Errorf("decode contact key file: %w", err)
	}
	if err := validateContactKeyFile(contact); err != nil {
		return contactKeyFile{}, err
	}
	return contact, nil
}

func saveContactKeyFile(path string, contact contactKeyFile) error {
	if err := validateContactKeyFile(contact); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return safePathError("create contact key file directory", err)
		}
	}
	body, err := json.MarshalIndent(contact, "", "  ")
	if err != nil {
		return fmt.Errorf("encode contact key file: %w", err)
	}
	body = append(body, '\n')
	if err := writeFileAtomicNoReplace(path, body, 0o600); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("write contact key file: output file already exists")
		}
		return safePathError("write contact key file", err)
	}
	return nil
}

func validateContactKeyFile(contact contactKeyFile) error {
	if contact.Version != contactKeyFileVersion {
		return fmt.Errorf("unsupported contact key file version")
	}
	if contact.Scope != wrappedKeyArtifactScope {
		return fmt.Errorf("unsupported contact key file scope")
	}
	if !validWrappedKeyContactID(contact.ContactID) {
		return fmt.Errorf("contact key file has invalid contact_id")
	}
	if contact.WrappingAlgorithm != wrappingAlgorithmAgeX25519 {
		return fmt.Errorf("unsupported contact key algorithm %q", contact.WrappingAlgorithm)
	}
	identity, err := age.ParseX25519Identity(contact.Identity)
	if err != nil {
		return fmt.Errorf("decode contact identity: %w", err)
	}
	if contact.Recipient == "" {
		return fmt.Errorf("contact recipient is required")
	}
	if _, err := age.ParseX25519Recipient(contact.Recipient); err != nil {
		return fmt.Errorf("decode contact recipient: %w", err)
	}
	if identity.Recipient().String() != contact.Recipient {
		return fmt.Errorf("contact identity does not match recipient")
	}
	if contact.ContactKeyID != contactKeyIDForRecipient(contact.Recipient) {
		return fmt.Errorf("contact key id does not match recipient")
	}
	return nil
}

func loadOrCreateWrappedKeyArtifact(path string, contact contactKeyFile, incidentID, streamID string, mediaKey envelope.Key) (wrappedKeyArtifact, bool, error) {
	artifact, err := loadWrappedKeyArtifact(path)
	if err == nil {
		if err := validateWrappedKeyArtifactForStream(artifact, incidentID, streamID, mediaKey.KeyID, contact.ContactKeyID); err != nil {
			return wrappedKeyArtifact{}, false, err
		}
		return artifact, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return wrappedKeyArtifact{}, false, safePathError("load wrapped-key artifact", err)
	}

	artifact, err = newWrappedKeyArtifact(contact, incidentID, streamID, mediaKey)
	if err != nil {
		return wrappedKeyArtifact{}, false, err
	}
	if err := saveWrappedKeyArtifact(path, artifact); err != nil {
		return wrappedKeyArtifact{}, false, err
	}
	return artifact, true, nil
}

func loadWrappedKeyArtifact(path string) (wrappedKeyArtifact, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return wrappedKeyArtifact{}, err
	}
	return decodeWrappedKeyArtifact(body)
}

func decodeWrappedKeyArtifact(body []byte) (wrappedKeyArtifact, error) {
	var artifact wrappedKeyArtifact
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&artifact); err != nil {
		return wrappedKeyArtifact{}, fmt.Errorf("decode wrapped-key artifact: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return wrappedKeyArtifact{}, fmt.Errorf("decode wrapped-key artifact: %w", err)
	}
	if err := validateWrappedKeyArtifact(artifact); err != nil {
		return wrappedKeyArtifact{}, err
	}
	return artifact, nil
}

func saveWrappedKeyArtifact(path string, artifact wrappedKeyArtifact) error {
	if err := validateWrappedKeyArtifact(artifact); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return safePathError("create wrapped-key artifact directory", err)
		}
	}
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("encode wrapped-key artifact: %w", err)
	}
	body = append(body, '\n')
	if err := writeFileAtomicNoReplace(path, body, 0o600); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("write wrapped-key artifact: output file already exists")
		}
		return safePathError("write wrapped-key artifact", err)
	}
	return nil
}

func newWrappedKeyArtifact(contact contactKeyFile, incidentID, streamID string, mediaKey envelope.Key) (wrappedKeyArtifact, error) {
	recipient, err := age.ParseX25519Recipient(contact.Recipient)
	if err != nil {
		return wrappedKeyArtifact{}, fmt.Errorf("decode contact recipient: %w", err)
	}
	wrapped, err := wrapMediaKeyForRecipient(mediaKey, recipient)
	if err != nil {
		return wrappedKeyArtifact{}, err
	}
	wrappedKeyID, err := randomWrappedKeyID()
	if err != nil {
		return wrappedKeyArtifact{}, err
	}
	artifact := wrappedKeyArtifact{
		Version:    wrappedKeyArtifactVersion,
		Scope:      wrappedKeyArtifactScope,
		IncidentID: incidentID,
		StreamID:   streamID,
		MediaKeyID: mediaKey.KeyID,
		CreatedAt:  time.Now().UTC(),
		WrappedKeys: []wrappedKeyRecord{
			{
				WrappedKeyID:      wrappedKeyID,
				RecipientType:     wrappedKeyRecipientType,
				ContactID:         contact.ContactID,
				ContactKeyID:      contact.ContactKeyID,
				WrappingAlgorithm: wrappingAlgorithmAgeX25519,
				WrappedKeyB64:     base64.RawURLEncoding.EncodeToString(wrapped),
			},
		},
	}
	if err := validateWrappedKeyArtifact(artifact); err != nil {
		return wrappedKeyArtifact{}, err
	}
	return artifact, nil
}

func wrapMediaKeyForRecipient(mediaKey envelope.Key, recipient age.Recipient) ([]byte, error) {
	if mediaKey.KeyID == "" || len(mediaKey.Key) != simulatorMediaKeySize {
		return nil, fmt.Errorf("simulator media key is invalid")
	}
	var body bytes.Buffer
	writer, err := age.Encrypt(&body, recipient)
	if err != nil {
		return nil, fmt.Errorf("wrap media key: %w", err)
	}
	if _, err := writer.Write(mediaKey.Key); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("write media key wrapper: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close media key wrapper: %w", err)
	}
	return body.Bytes(), nil
}

func unwrapWrappedMediaKey(artifact wrappedKeyArtifact, contact contactKeyFile) (envelope.Key, error) {
	if err := validateWrappedKeyArtifactForStream(artifact, artifact.IncidentID, artifact.StreamID, artifact.MediaKeyID, contact.ContactKeyID); err != nil {
		return envelope.Key{}, err
	}
	identity, err := age.ParseX25519Identity(contact.Identity)
	if err != nil {
		return envelope.Key{}, fmt.Errorf("decode contact identity: %w", err)
	}
	for _, record := range artifact.WrappedKeys {
		if record.ContactKeyID != contact.ContactKeyID {
			continue
		}
		wrapped, err := base64.RawURLEncoding.DecodeString(record.WrappedKeyB64)
		if err != nil {
			return envelope.Key{}, fmt.Errorf("decode wrapped media key: %w", err)
		}
		reader, err := age.Decrypt(bytes.NewReader(wrapped), identity)
		if err != nil {
			return envelope.Key{}, fmt.Errorf("unwrap media key: %w", err)
		}
		mediaKey, err := io.ReadAll(io.LimitReader(reader, simulatorMediaKeySize+1))
		if err != nil {
			return envelope.Key{}, fmt.Errorf("read unwrapped media key: %w", err)
		}
		if len(mediaKey) != simulatorMediaKeySize {
			return envelope.Key{}, fmt.Errorf("unwrapped media key has invalid size")
		}
		return envelope.Key{
			Version:   1,
			Scheme:    envelope.SchemeV1,
			Algorithm: envelope.AlgorithmAES256GCM,
			KeyID:     artifact.MediaKeyID,
			Key:       mediaKey,
		}, nil
	}
	return envelope.Key{}, fmt.Errorf("wrapped-key artifact has no record for contact key")
}

func validateWrappedKeyArtifactForStream(artifact wrappedKeyArtifact, incidentID, streamID, mediaKeyID, contactKeyID string) error {
	if err := validateWrappedKeyArtifact(artifact); err != nil {
		return err
	}
	if artifact.IncidentID != incidentID || artifact.StreamID != streamID || artifact.MediaKeyID != mediaKeyID {
		return fmt.Errorf("wrapped-key artifact does not match stream")
	}
	for _, record := range artifact.WrappedKeys {
		if record.ContactKeyID == contactKeyID {
			return nil
		}
	}
	return fmt.Errorf("wrapped-key artifact has no record for contact key")
}

func validateWrappedKeyArtifact(artifact wrappedKeyArtifact) error {
	if artifact.Version != wrappedKeyArtifactVersion {
		return fmt.Errorf("unsupported wrapped-key artifact version")
	}
	if artifact.Scope != wrappedKeyArtifactScope {
		return fmt.Errorf("unsupported wrapped-key artifact scope")
	}
	if strings.TrimSpace(artifact.IncidentID) == "" || strings.TrimSpace(artifact.StreamID) == "" || strings.TrimSpace(artifact.MediaKeyID) == "" {
		return fmt.Errorf("wrapped-key artifact is missing incident, stream, or media key identity")
	}
	if len(artifact.WrappedKeys) == 0 {
		return fmt.Errorf("wrapped-key artifact has no wrapped keys")
	}
	for _, record := range artifact.WrappedKeys {
		if strings.TrimSpace(record.WrappedKeyID) == "" {
			return fmt.Errorf("wrapped-key artifact has missing wrapped_key_id")
		}
		if record.RecipientType != wrappedKeyRecipientType {
			return fmt.Errorf("unsupported wrapped-key recipient type %q", record.RecipientType)
		}
		if !validWrappedKeyContactID(record.ContactID) {
			return fmt.Errorf("wrapped-key artifact has invalid contact_id")
		}
		if strings.TrimSpace(record.ContactKeyID) == "" {
			return fmt.Errorf("wrapped-key artifact has missing contact_key_id")
		}
		if record.WrappingAlgorithm != wrappingAlgorithmAgeX25519 {
			return fmt.Errorf("unsupported wrapped-key algorithm %q", record.WrappingAlgorithm)
		}
		if strings.TrimSpace(record.WrappedKeyB64) == "" {
			return fmt.Errorf("wrapped-key artifact has missing wrapped_key_b64")
		}
		if _, err := base64.RawURLEncoding.DecodeString(record.WrappedKeyB64); err != nil {
			return fmt.Errorf("decode wrapped_key_b64: %w", err)
		}
	}
	return nil
}

func contactKeyIDForRecipient(recipient string) string {
	sum := sha256.Sum256([]byte("proofline-sim-contact-key-id-v1\n" + recipient))
	return "ckid_" + base64.RawURLEncoding.EncodeToString(sum[:16])
}

func randomWrappedKeyID() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generate wrapped key id: %w", err)
	}
	return "wkey_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func validWrappedKeyContactID(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var trailing struct{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON")
		}
		return err
	}
	return nil
}
