package envelope

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// ParseHeader parses and validates the non-secret envelope header.
func ParseHeader(envelopeBytes []byte) (Header, error) {
	header, _, err := parseEnvelope(envelopeBytes)
	return header, err
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
