package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// EncryptChunk wraps plaintext in the v1 compatibility AES-256-GCM chunk envelope.
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

// DecryptChunk opens the v1 compatibility chunk envelope using the expected chunk metadata.
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
