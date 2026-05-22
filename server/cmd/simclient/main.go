package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"safety-recorder/server/internal/envelope"
)

func main() {
	if err := run(context.Background(), os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "simclient: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer, args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}

	encryptionKey, err := prepareEncryption(out, cfg)
	if err != nil {
		return err
	}

	sim := client{
		httpClient: &http.Client{Timeout: clientRequestTimeout},
		apiBase:    cfg.apiBase,
		viewerBase: cfg.viewerBase,
	}

	fmt.Fprintln(out, "Creating incident...")
	incidentID, err := sim.createIncident(ctx)
	if err != nil {
		return err
	}

	token, err := sim.createEmergencyToken(ctx, incidentID)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Incident: %s\n", incidentID)
	fmt.Fprintf(out, "Emergency viewer: %s\n\n", buildViewerURL(sim.viewerBase, token))

	fmt.Fprintf(out, "Creating %s media stream...\n", cfg.mediaType)
	streamID, err := sim.createMediaStream(ctx, incidentID, cfg.mediaType)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Stream: %s\n\n", streamID)

	if err := uploadChunks(ctx, out, sim, cfg, incidentID, streamID, encryptionKey); err != nil {
		return err
	}

	if cfg.completeStream && cfg.chunks > 0 {
		fmt.Fprintln(out, "Completing stream...")
		if err := sim.completeMediaStream(ctx, incidentID, streamID, cfg.chunks); err != nil {
			return err
		}
		fmt.Fprintln(out, "Stream complete.")
	}

	if cfg.downloadBundle {
		if err := downloadAndVerifyBundle(ctx, out, sim, cfg, token, incidentID, streamID, encryptionKey); err != nil {
			return err
		}
	}

	if cfg.closeIncident {
		fmt.Fprintln(out, "Closing incident...")
		if err := sim.closeIncident(ctx, incidentID); err != nil {
			return err
		}
		fmt.Fprintln(out, "Incident closed.")
	}

	fmt.Fprintln(out, "Done.")
	return nil
}

func prepareEncryption(out io.Writer, cfg config) (envelope.Key, error) {
	if !cfg.encrypt {
		fmt.Fprintln(out, "Encryption: disabled. Uploading raw fake chunk bytes for development compatibility only.")
		fmt.Fprintln(out)
		return envelope.Key{}, nil
	}

	encryptionKey, err := loadOrCreateSimulatorKey(cfg.keyFile)
	if err != nil {
		return envelope.Key{}, err
	}
	fmt.Fprintln(out, "Encryption: enabled")
	fmt.Fprintf(out, "Key ID: %s\n", encryptionKey.KeyID)
	if cfg.keyFile != "" {
		fmt.Fprintf(out, "Key file: %s\n", cfg.keyFile)
	}
	fmt.Fprintln(out)
	return encryptionKey, nil
}

func uploadChunks(ctx context.Context, out io.Writer, sim client, cfg config, incidentID, streamID string, key envelope.Key) error {
	startedAt := time.Now().UTC()
	for i := 1; i <= cfg.chunks; i++ {
		chunk, err := createUploadChunk(cfg, key, incidentID, streamID, i, startedAt)
		if err != nil {
			return err
		}
		if err := uploadChunkWithOptionalHashFailure(ctx, out, sim, cfg, chunk, i); err != nil {
			return err
		}
		if shouldSendCheckin(i) {
			fmt.Fprintln(out, "Sending checkin...")
			if err := sim.createCheckin(ctx, incidentID, i); err != nil {
				return err
			}
		}
		if i < cfg.chunks && cfg.interval > 0 {
			time.Sleep(cfg.interval)
		}
	}
	return nil
}

func createUploadChunk(cfg config, key envelope.Key, incidentID, streamID string, chunkIndex int, startedAt time.Time) (chunkUpload, error) {
	if cfg.encrypt {
		return newEncryptedChunkUpload(key, incidentID, streamID, chunkIndex, cfg.mediaType, cfg.chunkSize, startedAt)
	}
	return newChunkUpload(incidentID, streamID, chunkIndex, cfg.mediaType, cfg.chunkSize, startedAt)
}

func uploadChunkWithOptionalHashFailure(ctx context.Context, out io.Writer, sim client, cfg config, chunk chunkUpload, chunkIndex int) error {
	if !shouldSimulateFailure(chunkIndex, cfg.simulateFailureEvery) {
		fmt.Fprintf(out, "Uploading %s%s chunk %d/%d...\n", encryptionLogPrefix(cfg.encrypt), cfg.mediaType, chunkIndex, cfg.chunks)
		return sim.uploadChunk(ctx, chunk)
	}

	fmt.Fprintf(out, "Uploading %s%s chunk %d/%d with intentionally bad hash...\n", encryptionLogPrefix(cfg.encrypt), cfg.mediaType, chunkIndex, cfg.chunks)
	failed := chunk
	failed.sha256Hex = badHashFor(chunk.sha256Hex)
	if err := sim.expectHashMismatch(ctx, failed); err != nil {
		return err
	}
	fmt.Fprintln(out, "Server rejected chunk as expected.")

	fmt.Fprintf(out, "Retrying %s%s chunk %d/%d with correct hash...\n", encryptionLogPrefix(cfg.encrypt), cfg.mediaType, chunkIndex, cfg.chunks)
	if err := sim.uploadChunk(ctx, chunk); err != nil {
		return err
	}
	fmt.Fprintln(out, "Retry succeeded.")
	return nil
}

func downloadAndVerifyBundle(ctx context.Context, out io.Writer, sim client, cfg config, token, incidentID, streamID string, key envelope.Key) error {
	fmt.Fprintln(out, "Testing emergency stream bundle download...")
	bundleBytes, err := sim.downloadStreamBundle(ctx, token, streamID)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Downloaded bundle.")
	if !cfg.encrypt || !cfg.verifyBundleDecrypt {
		fmt.Fprintln(out, "Bundle download succeeded.")
		return nil
	}

	verified, err := verifyStreamBundleDecryption(bundleBytes, key, incidentID, streamID, cfg.mediaType)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Verified decrypt of %d encrypted chunks.\n", verified)
	return nil
}
