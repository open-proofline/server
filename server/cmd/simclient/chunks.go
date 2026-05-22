package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"safety-recorder/server/internal/envelope"
)

type chunkUpload struct {
	incidentID string
	streamID   string
	chunkIndex int
	mediaType  string
	startedAt  time.Time
	endedAt    time.Time
	filename   string
	body       []byte
	sha256Hex  string
}

func newChunkUpload(incidentID, streamID string, chunkIndex int, mediaType string, size int64, startedAt time.Time) (chunkUpload, error) {
	body, err := randomChunkBytes(size)
	if err != nil {
		return chunkUpload{}, err
	}
	return buildChunkUpload(incidentID, streamID, chunkIndex, mediaType, startedAt, body), nil
}

func newEncryptedChunkUpload(key envelope.Key, incidentID, streamID string, chunkIndex int, mediaType string, size int64, startedAt time.Time) (chunkUpload, error) {
	plaintext, err := randomChunkBytes(size)
	if err != nil {
		return chunkUpload{}, err
	}
	body, err := envelope.EncryptChunk(key, chunkContext(incidentID, streamID, mediaType, chunkIndex), plaintext)
	if err != nil {
		return chunkUpload{}, fmt.Errorf("encrypt chunk: %w", err)
	}
	return buildChunkUpload(incidentID, streamID, chunkIndex, mediaType, startedAt, body), nil
}

func randomChunkBytes(size int64) ([]byte, error) {
	if size > int64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("chunk size is too large for this platform")
	}
	body := make([]byte, int(size))
	if _, err := rand.Read(body); err != nil {
		return nil, fmt.Errorf("generate fake chunk bytes: %w", err)
	}
	return body, nil
}

func buildChunkUpload(incidentID, streamID string, chunkIndex int, mediaType string, startedAt time.Time, body []byte) chunkUpload {
	sum := sha256.Sum256(body)
	chunkStartedAt := startedAt.Add(time.Duration(chunkIndex-1) * chunkDuration)
	return chunkUpload{
		incidentID: incidentID,
		streamID:   streamID,
		chunkIndex: chunkIndex,
		mediaType:  mediaType,
		startedAt:  chunkStartedAt,
		endedAt:    chunkStartedAt.Add(chunkDuration),
		filename:   fmt.Sprintf("%s_%06d.enc", mediaType, chunkIndex),
		body:       body,
		sha256Hex:  hex.EncodeToString(sum[:]),
	}
}

func loadOrCreateSimulatorKey(path string) (envelope.Key, error) {
	if path == "" {
		key, err := envelope.GenerateKey()
		if err != nil {
			return envelope.Key{}, fmt.Errorf("generate ephemeral encryption key: %w", err)
		}
		return key, nil
	}

	key, err := envelope.LoadKeyFile(path)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return envelope.Key{}, fmt.Errorf("load key file: %w", err)
	}
	key, err = envelope.GenerateKey()
	if err != nil {
		return envelope.Key{}, fmt.Errorf("generate encryption key: %w", err)
	}
	if err := envelope.SaveKeyFile(path, key); err != nil {
		return envelope.Key{}, fmt.Errorf("save key file: %w", err)
	}
	return key, nil
}

func chunkContext(incidentID, streamID, mediaType string, chunkIndex int) envelope.ChunkContext {
	return envelope.ChunkContext{
		IncidentID: incidentID,
		StreamID:   streamID,
		MediaType:  mediaType,
		ChunkIndex: chunkIndex,
	}
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func shouldSimulateFailure(chunkIndex, every int) bool {
	return every > 0 && chunkIndex%every == 0
}

func shouldSendCheckin(chunkIndex int) bool {
	return chunkIndex == 1 || chunkIndex%defaultCheckinEvery == 0
}

func encryptionLogPrefix(enabled bool) string {
	if enabled {
		return "encrypted "
	}
	return ""
}

func badHashFor(hash string) string {
	if len(hash) != 64 {
		return strings.Repeat("0", 64)
	}
	if hash[0] == '0' {
		return "1" + hash[1:]
	}
	return "0" + hash[1:]
}

func validMediaType(mediaType string) bool {
	switch mediaType {
	case "audio", "video", "location", "metadata":
		return true
	default:
		return false
	}
}
