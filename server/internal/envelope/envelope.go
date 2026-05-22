package envelope

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	SchemeV1           = "safety-recorder-chunk-encryption-v1"
	AlgorithmAES256GCM = "AES-256-GCM"

	keyVersion      = 1
	keySize         = 32
	nonceSize       = 12
	maxHeaderLength = 16 * 1024

	magic = "SRCENC1\n"
)

// ChunkContext is the metadata bound to a chunk through AES-GCM associated data.
type ChunkContext struct {
	IncidentID string
	StreamID   string
	MediaType  string
	ChunkIndex int
}

// Key is a simulator/development AES-256-GCM key.
type Key struct {
	Version   int
	Scheme    string
	Algorithm string
	KeyID     string
	Key       []byte
}

// Header is the non-secret JSON header stored inside an encrypted chunk envelope.
type Header struct {
	Version   int    `json:"version"`
	Scheme    string `json:"scheme"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	NonceB64  string `json:"nonce_b64"`
	AAD       string `json:"aad"`
}

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

// BuildAssociatedData returns the exact UTF-8 AEAD associated data for a chunk.
func BuildAssociatedData(ctx ChunkContext) ([]byte, error) {
	if err := rejectNewline("incident_id", ctx.IncidentID); err != nil {
		return nil, err
	}
	if err := rejectNewline("stream_id", ctx.StreamID); err != nil {
		return nil, err
	}
	if err := rejectNewline("media_type", ctx.MediaType); err != nil {
		return nil, err
	}
	if ctx.ChunkIndex <= 0 {
		return nil, fmt.Errorf("chunk_index must be positive")
	}
	return []byte(fmt.Sprintf(
		"SafetyRecorderChunk:v1\nincident_id=%s\nstream_id=%s\nmedia_type=%s\nchunk_index=%d\n",
		ctx.IncidentID,
		ctx.StreamID,
		ctx.MediaType,
		ctx.ChunkIndex,
	)), nil
}

// EncryptChunk wraps plaintext in a Safety Recorder v1 AES-256-GCM chunk envelope.
func EncryptChunk(key Key, ctx ChunkContext, plaintext []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	aad, err := BuildAssociatedData(ctx)
	if err != nil {
		return nil, err
	}
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	header := Header{
		Version:   keyVersion,
		Scheme:    SchemeV1,
		Algorithm: AlgorithmAES256GCM,
		KeyID:     key.KeyID,
		NonceB64:  base64.RawURLEncoding.EncodeToString(nonce),
		AAD:       string(aad),
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}
	if len(headerBytes) > maxHeaderLength {
		return nil, fmt.Errorf("header length exceeds %d bytes", maxHeaderLength)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, aad)
	envelopeBytes := make([]byte, 0, len(magic)+4+len(headerBytes)+len(ciphertext))
	envelopeBytes = append(envelopeBytes, []byte(magic)...)
	headerLength := make([]byte, 4)
	binary.BigEndian.PutUint32(headerLength, uint32(len(headerBytes)))
	envelopeBytes = append(envelopeBytes, headerLength...)
	envelopeBytes = append(envelopeBytes, headerBytes...)
	envelopeBytes = append(envelopeBytes, ciphertext...)
	return envelopeBytes, nil
}

// DecryptChunk opens a Safety Recorder v1 chunk envelope using the expected chunk metadata.
func DecryptChunk(key Key, ctx ChunkContext, envelopeBytes []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	header, ciphertext, err := parseEnvelope(envelopeBytes)
	if err != nil {
		return nil, err
	}
	if header.KeyID != key.KeyID {
		return nil, fmt.Errorf("key_id does not match header")
	}
	expectedAAD, err := BuildAssociatedData(ctx)
	if err != nil {
		return nil, err
	}
	if header.AAD != string(expectedAAD) {
		return nil, fmt.Errorf("associated data does not match expected metadata")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(header.NonceB64)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, expectedAAD)
	if err != nil {
		return nil, fmt.Errorf("decrypt chunk: %w", err)
	}
	return plaintext, nil
}

// ParseHeader parses and validates the non-secret envelope header.
func ParseHeader(envelopeBytes []byte) (Header, error) {
	header, _, err := parseEnvelope(envelopeBytes)
	return header, err
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

func parseEnvelope(envelopeBytes []byte) (Header, []byte, error) {
	if len(envelopeBytes) < len(magic)+4 {
		return Header{}, nil, fmt.Errorf("envelope is truncated")
	}
	if !bytes.HasPrefix(envelopeBytes, []byte(magic)) {
		return Header{}, nil, fmt.Errorf("invalid magic")
	}
	headerLength := binary.BigEndian.Uint32(envelopeBytes[len(magic) : len(magic)+4])
	if headerLength == 0 {
		return Header{}, nil, fmt.Errorf("header is missing")
	}
	if headerLength > maxHeaderLength {
		return Header{}, nil, fmt.Errorf("header length exceeds %d bytes", maxHeaderLength)
	}
	headerStart := len(magic) + 4
	headerEnd := headerStart + int(headerLength)
	if len(envelopeBytes) < headerEnd {
		return Header{}, nil, fmt.Errorf("envelope header is truncated")
	}
	headerBytes := envelopeBytes[headerStart:headerEnd]
	if !utf8.Valid(headerBytes) {
		return Header{}, nil, fmt.Errorf("header is not valid UTF-8")
	}
	ciphertext := envelopeBytes[headerEnd:]
	if len(ciphertext) == 0 {
		return Header{}, nil, fmt.Errorf("ciphertext is missing")
	}
	header, err := parseHeaderJSON(headerBytes)
	if err != nil {
		return Header{}, nil, err
	}
	return header, ciphertext, nil
}

func parseHeaderJSON(headerBytes []byte) (Header, error) {
	decoder := json.NewDecoder(bytes.NewReader(headerBytes))
	decoder.DisallowUnknownFields()
	var header Header
	if err := decoder.Decode(&header); err != nil {
		return Header{}, fmt.Errorf("decode header: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return Header{}, fmt.Errorf("decode header: %w", err)
	}
	if header.Version != keyVersion {
		return Header{}, fmt.Errorf("unsupported envelope version %d", header.Version)
	}
	if header.Scheme != SchemeV1 {
		return Header{}, fmt.Errorf("unsupported envelope scheme %q", header.Scheme)
	}
	if header.Algorithm != AlgorithmAES256GCM {
		return Header{}, fmt.Errorf("unsupported envelope algorithm %q", header.Algorithm)
	}
	if header.KeyID == "" {
		return Header{}, fmt.Errorf("key_id is required")
	}
	if header.NonceB64 == "" {
		return Header{}, fmt.Errorf("nonce_b64 is required")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(header.NonceB64)
	if err != nil {
		return Header{}, fmt.Errorf("decode nonce: %w", err)
	}
	if len(nonce) != nonceSize {
		return Header{}, fmt.Errorf("nonce must be %d bytes", nonceSize)
	}
	if header.AAD == "" {
		return Header{}, fmt.Errorf("aad is required")
	}
	return header, nil
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

func newAEAD(key Key) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	if aead.NonceSize() != nonceSize {
		return nil, fmt.Errorf("unexpected GCM nonce size %d", aead.NonceSize())
	}
	return aead, nil
}

func rejectNewline(field, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s must not contain newlines", field)
	}
	return nil
}
