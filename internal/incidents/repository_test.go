package incidents_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/open-proofline/server/internal/db"
	"github.com/open-proofline/server/internal/incidents"
)

func TestCreateChunkRejectsClosedIncident(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	if _, err := repo.CloseIncident(ctx, incident.ID); err != nil {
		t.Fatalf("close incident: %v", err)
	}

	_, err = repo.CreateChunk(ctx, testChunkParams(incident.ID, "", incidents.MediaTypeAudio, 1))

	if !errors.Is(err, incidents.ErrIncidentClosed) {
		t.Fatalf("expected ErrIncidentClosed, got %v", err)
	}
	chunks, err := repo.ListChunks(ctx, incident.ID)
	if err != nil {
		t.Fatalf("list chunks: %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected no chunks, got %+v", chunks)
	}
}

func TestCreateChunkRejectsCompletedStream(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	stream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "audio")
	if err != nil {
		t.Fatalf("create media stream: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, stream.ID, incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create stream chunk: %v", err)
	}
	if _, err := repo.CompleteMediaStream(ctx, incident.ID, stream.ID, 1); err != nil {
		t.Fatalf("complete media stream: %v", err)
	}

	_, err = repo.CreateChunk(ctx, testChunkParams(incident.ID, stream.ID, incidents.MediaTypeAudio, 2))

	if !errors.Is(err, incidents.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
	chunks, err := repo.ListStreamChunks(ctx, incident.ID, stream.ID)
	if err != nil {
		t.Fatalf("list stream chunks: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected only the original stream chunk, got %+v", chunks)
	}
}

func TestCreateChunkUsesStreamScopedDuplicateIdentity(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	firstStream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "first audio")
	if err != nil {
		t.Fatalf("create first media stream: %v", err)
	}
	secondStream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "second audio")
	if err != nil {
		t.Fatalf("create second media stream: %v", err)
	}

	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, firstStream.ID, incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create first stream chunk: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, secondStream.ID, incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create second stream chunk with same index: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, "", incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create legacy chunk with same media/index: %v", err)
	}

	_, err = repo.CreateChunk(ctx, testChunkParams(incident.ID, firstStream.ID, incidents.MediaTypeAudio, 1))
	if !errors.Is(err, incidents.ErrDuplicate) {
		t.Fatalf("expected duplicate stream chunk to return ErrDuplicate, got %v", err)
	}
}

func TestGetChunkByKeyReturnsLegacyUnstreamedChunk(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	stream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "audio")
	if err != nil {
		t.Fatalf("create media stream: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, stream.ID, incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create stream chunk: %v", err)
	}
	legacy, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, "", incidents.MediaTypeAudio, 1))
	if err != nil {
		t.Fatalf("create legacy chunk: %v", err)
	}

	got, err := repo.GetChunkByKey(ctx, incident.ID, incidents.MediaTypeAudio, 1)
	if err != nil {
		t.Fatalf("get legacy chunk by key: %v", err)
	}
	if got.ID != legacy.ID || got.StreamID != "" {
		t.Fatalf("expected legacy chunk %+v, got %+v", legacy, got)
	}
}

func TestUploadOperationReservationCompletionAndConflict(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	stream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "audio")
	if err != nil {
		t.Fatalf("create media stream: %v", err)
	}
	params := testUploadOperationParams(incident.ID, stream.ID)

	reserved, err := repo.ReserveUploadOperation(ctx, params)
	if err != nil {
		t.Fatalf("reserve upload operation: %v", err)
	}
	if reserved.State != incidents.UploadOperationStateReserved {
		t.Fatalf("state = %q, want reserved", reserved.State)
	}
	same, err := repo.ReserveUploadOperation(ctx, params)
	if err != nil {
		t.Fatalf("reserve same upload operation: %v", err)
	}
	if same.ID != reserved.ID {
		t.Fatalf("same idempotency key created a new operation: first=%q second=%q", reserved.ID, same.ID)
	}

	conflicting := params
	conflicting.OriginalFilename = "other.enc"
	conflicting.FingerprintHash = strings.Repeat("c", 64)
	if _, err := repo.ReserveUploadOperation(ctx, conflicting); !errors.Is(err, incidents.ErrIdempotencyConflict) {
		t.Fatalf("conflicting reservation error = %v, want ErrIdempotencyConflict", err)
	}

	chunk, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, stream.ID, incidents.MediaTypeAudio, 1))
	if err != nil {
		t.Fatalf("create chunk: %v", err)
	}
	completed, err := repo.CompleteUploadOperation(ctx, params, chunk)
	if err != nil {
		t.Fatalf("complete upload operation: %v", err)
	}
	if completed.State != incidents.UploadOperationStateMetadataCommitted || completed.ChunkID != chunk.ID {
		t.Fatalf("unexpected completed operation: %+v", completed)
	}
	replayed, err := repo.ReserveUploadOperation(ctx, params)
	if err != nil {
		t.Fatalf("reserve completed operation: %v", err)
	}
	if replayed.State != incidents.UploadOperationStateMetadataCommitted || replayed.ChunkID != chunk.ID {
		t.Fatalf("expected completed operation replay, got %+v", replayed)
	}
}

func TestCompleteMediaStreamRejectsUnexpectedChunkRows(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	stream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "audio")
	if err != nil {
		t.Fatalf("create media stream: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, stream.ID, incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create first stream chunk: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, stream.ID, incidents.MediaTypeAudio, 2)); err != nil {
		t.Fatalf("create second stream chunk: %v", err)
	}

	_, err = repo.CompleteMediaStream(ctx, incident.ID, stream.ID, 1)

	if !errors.Is(err, incidents.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
	stream, err = repo.GetMediaStream(ctx, incident.ID, stream.ID)
	if err != nil {
		t.Fatalf("get media stream: %v", err)
	}
	if stream.Status != incidents.StreamStatusOpen {
		t.Fatalf("expected stream to remain open, got %+v", stream)
	}
}

func newRepository(t *testing.T, ctx context.Context) *incidents.Repository {
	t.Helper()

	conn, err := db.Open(ctx, filepath.Join(t.TempDir(), "safety.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return incidents.NewRepository(conn)
}

func testChunkParams(incidentID, streamID, mediaType string, chunkIndex int) incidents.CreateChunkParams {
	startedAt := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	storedPath := fmt.Sprintf("incidents/%s/%s_%06d.enc", incidentID, mediaType, chunkIndex)
	if streamID != "" {
		storedPath = fmt.Sprintf("incidents/%s/streams/%s/%s_%06d.enc", incidentID, streamID, mediaType, chunkIndex)
	}
	return incidents.CreateChunkParams{
		IncidentID:       incidentID,
		StreamID:         streamID,
		ChunkIndex:       chunkIndex,
		MediaType:        mediaType,
		StartedAt:        startedAt,
		EndedAt:          startedAt.Add(time.Second),
		OriginalFilename: "chunk.enc",
		StoredPath:       storedPath,
		ByteSize:         4,
		SHA256Hex:        strings.Repeat("a", 64),
	}
}

func testUploadOperationParams(incidentID, streamID string) incidents.UploadOperationParams {
	chunk := testChunkParams(incidentID, streamID, incidents.MediaTypeAudio, 1)
	return incidents.UploadOperationParams{
		Operation:          incidents.UploadOperationUploadChunk,
		IdempotencyKeyHash: strings.Repeat("b", 64),
		IncidentID:         incidentID,
		StreamID:           streamID,
		ChunkIndex:         chunk.ChunkIndex,
		MediaType:          chunk.MediaType,
		StartedAt:          chunk.StartedAt,
		EndedAt:            chunk.EndedAt,
		OriginalFilename:   chunk.OriginalFilename,
		ByteSize:           chunk.ByteSize,
		SHA256Hex:          chunk.SHA256Hex,
		FingerprintHash:    strings.Repeat("a", 64),
	}
}
