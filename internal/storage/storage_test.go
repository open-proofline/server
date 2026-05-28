package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLocalStoreBlobStoreSemantics(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	var blobStore BlobStore = store
	ctx := context.Background()

	payload := "encrypted audio data"
	upload, err := blobStore.SaveTemp(ctx, strings.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("save temp: %v", err)
	}
	if upload.ByteSize != int64(len(payload)) || upload.SHA256Hex != sha256Hex(payload) {
		t.Fatalf("unexpected staged upload metadata: %+v", upload)
	}

	storedPath, err := blobStore.CommitTemp(ctx, upload, "inc_test", "str_test", "audio", 1)
	if err != nil {
		t.Fatalf("commit temp: %v", err)
	}
	if storedPath != "incidents/inc_test/streams/str_test/audio_000001.enc" {
		t.Fatalf("stored path = %q", storedPath)
	}
	if upload.Path != "" {
		t.Fatalf("expected committed upload cleanup, got temp path %q", upload.Path)
	}

	reader, err := blobStore.Open(ctx, storedPath)
	if err != nil {
		t.Fatalf("open stored blob: %v", err)
	}
	stored, err := io.ReadAll(reader)
	if closeErr := reader.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatalf("read stored blob: %v", err)
	}
	if string(stored) != payload {
		t.Fatalf("stored payload mismatch: %q", stored)
	}

	duplicateUpload, err := blobStore.SaveTemp(ctx, strings.NewReader("replacement"), int64(len("replacement")))
	if err != nil {
		t.Fatalf("save duplicate temp: %v", err)
	}
	t.Cleanup(duplicateUpload.Cleanup)
	_, err = blobStore.CommitTemp(ctx, duplicateUpload, "inc_test", "str_test", "audio", 1)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate commit error = %v, want ErrAlreadyExists", err)
	}

	reader, err = blobStore.Open(ctx, storedPath)
	if err != nil {
		t.Fatalf("open original after duplicate: %v", err)
	}
	stored, err = io.ReadAll(reader)
	if closeErr := reader.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatalf("read original after duplicate: %v", err)
	}
	if string(stored) != payload {
		t.Fatalf("duplicate commit overwrote stored payload: %q", stored)
	}

	if err := blobStore.Remove(ctx, storedPath); err != nil {
		t.Fatalf("remove stored blob: %v", err)
	}
	_, err = blobStore.Open(ctx, storedPath)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("open removed blob error = %v, want os.ErrNotExist", err)
	}
}

func TestLocalStoreRejectsUnsafeStoredPaths(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()

	for _, storedPath := range []string{
		"",
		"/absolute",
		"../escape",
		"incidents/inc_test/../escape",
		"incidents//inc_test/audio_000001.enc",
		"incidents\\inc_test\\audio_000001.enc",
	} {
		if _, err := store.Open(ctx, storedPath); !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("open %q error = %v, want ErrUnsafePath", storedPath, err)
		}
		if err := store.Remove(ctx, storedPath); !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("remove %q error = %v, want ErrUnsafePath", storedPath, err)
		}
	}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
