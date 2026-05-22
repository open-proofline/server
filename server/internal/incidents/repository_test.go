package incidents_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"safety-recorder/server/internal/db"
	"safety-recorder/server/internal/incidents"
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
	return incidents.CreateChunkParams{
		IncidentID:       incidentID,
		StreamID:         streamID,
		ChunkIndex:       chunkIndex,
		MediaType:        mediaType,
		StartedAt:        startedAt,
		EndedAt:          startedAt.Add(time.Second),
		OriginalFilename: "chunk.enc",
		StoredPath:       fmt.Sprintf("incidents/%s/%s_%06d.enc", incidentID, mediaType, chunkIndex),
		ByteSize:         4,
		SHA256Hex:        strings.Repeat("a", 64),
	}
}
