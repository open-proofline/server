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
	"github.com/open-proofline/server/internal/incidents/contracttest"
)

func TestCreateIncidentStoresModeFields(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)

	incident, err := repo.CreateIncidentForAccount(ctx, "", incidents.CreateIncidentParams{
		ClientLabel:      "phone",
		IncidentMode:     incidents.IncidentModeSafetyCheck,
		CaptureProfile:   incidents.CaptureProfileLocationCheckin,
		EscalationPolicy: incidents.EscalationPolicyTrustedContactsOnMissedCheckin,
		SharingState:     incidents.SharingStatePrivate,
	})
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	got, err := repo.GetIncident(ctx, incident.ID)
	if err != nil {
		t.Fatalf("get incident: %v", err)
	}
	if got.IncidentMode != incidents.IncidentModeSafetyCheck ||
		got.CaptureProfile != incidents.CaptureProfileLocationCheckin ||
		got.EscalationPolicy != incidents.EscalationPolicyTrustedContactsOnMissedCheckin ||
		got.SharingState != incidents.SharingStatePrivate {
		t.Fatalf("incident mode fields were not preserved: %+v", got)
	}
}

func TestSQLiteUploadOperationRaceAndBackendParity(t *testing.T) {
	contracttest.RunUploadOperationRaceAndParity(t, func(t *testing.T, ctx context.Context) contracttest.Repository {
		t.Helper()
		return newRepository(t, ctx)
	})
}

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

func TestRequestIncidentDeletionSnapshotsItemsAndBlocksWrites(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncidentForAccount(ctx, "", incidents.CreateIncidentParams{
		ClientLabel: "phone",
		Notes:       "delete me",
	})
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
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, "", incidents.MediaTypeMetadata, 1)); err != nil {
		t.Fatalf("create metadata chunk: %v", err)
	}

	status, err := repo.RequestIncidentDeletion(ctx, incidents.IncidentDeletionRequest{
		IncidentID:     incident.ID,
		Source:         incidents.IncidentDeletionSourceAccountRequest,
		ReasonCode:     "account_delete",
		ActorAccountID: "acct_owner",
		AllowOpen:      true,
	})
	if err != nil {
		t.Fatalf("request deletion: %v", err)
	}
	if status.State != incidents.IncidentDeletionStatePending || status.ItemCount != 2 {
		t.Fatalf("unexpected deletion status: %+v", status)
	}

	got, err := repo.GetIncident(ctx, incident.ID)
	if err != nil {
		t.Fatalf("get incident: %v", err)
	}
	if got.DeletionState != incidents.IncidentDeletionStatePending {
		t.Fatalf("deletion state = %q, want pending", got.DeletionState)
	}
	items, err := repo.ListIncidentDeletionItems(ctx, status.DecisionID)
	if err != nil {
		t.Fatalf("list deletion items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("deletion item count = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.StoredPath == "" || item.State != incidents.IncidentDeletionItemStatePending {
			t.Fatalf("unexpected deletion item: %+v", item)
		}
	}

	repeated, err := repo.RequestIncidentDeletion(ctx, incidents.IncidentDeletionRequest{
		IncidentID:     incident.ID,
		Source:         incidents.IncidentDeletionSourceAccountRequest,
		ActorAccountID: "acct_owner",
		AllowOpen:      true,
	})
	if err != nil {
		t.Fatalf("repeat deletion request: %v", err)
	}
	if repeated.DecisionID != status.DecisionID || repeated.ItemCount != status.ItemCount {
		t.Fatalf("repeat request returned %+v, want existing %+v", repeated, status)
	}

	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, "", incidents.MediaTypeAudio, 2)); !errors.Is(err, incidents.ErrIncidentDeleting) {
		t.Fatalf("create chunk during deletion error = %v, want ErrIncidentDeleting", err)
	}
	if _, err := repo.CreateCheckin(ctx, incident.ID, incidents.CreateCheckinParams{}); !errors.Is(err, incidents.ErrIncidentDeleting) {
		t.Fatalf("create checkin during deletion error = %v, want ErrIncidentDeleting", err)
	}
	if _, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "late audio"); !errors.Is(err, incidents.ErrIncidentDeleting) {
		t.Fatalf("create stream during deletion error = %v, want ErrIncidentDeleting", err)
	}
	if _, err := repo.CompleteMediaStream(ctx, incident.ID, stream.ID, 1); !errors.Is(err, incidents.ErrIncidentDeleting) {
		t.Fatalf("complete stream during deletion error = %v, want ErrIncidentDeleting", err)
	}
	if _, err := repo.FailMediaStream(ctx, incident.ID, stream.ID, "late failure"); !errors.Is(err, incidents.ErrIncidentDeleting) {
		t.Fatalf("fail stream during deletion error = %v, want ErrIncidentDeleting", err)
	}
	if _, err := repo.CloseIncident(ctx, incident.ID); !errors.Is(err, incidents.ErrIncidentDeleting) {
		t.Fatalf("close incident during deletion error = %v, want ErrIncidentDeleting", err)
	}
	if _, err := repo.ReserveUploadOperation(ctx, testUploadOperationParams(incident.ID, stream.ID)); !errors.Is(err, incidents.ErrIncidentDeleting) {
		t.Fatalf("reserve upload operation during deletion error = %v, want ErrIncidentDeleting", err)
	}
	if _, _, err := repo.CreateIncidentToken(ctx, incident.ID, "trusted contact", nil); !errors.Is(err, incidents.ErrNotFound) {
		t.Fatalf("create token during deletion error = %v, want ErrNotFound", err)
	}
}

func TestRequestIncidentDeletionRejectsOpenIncidentWithoutAllowOpen(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	_, err = repo.RequestIncidentDeletion(ctx, incidents.IncidentDeletionRequest{
		IncidentID: incident.ID,
		Source:     incidents.IncidentDeletionSourceAccountRequest,
	})
	if !errors.Is(err, incidents.ErrInvalidState) {
		t.Fatalf("open deletion error = %v, want ErrInvalidState", err)
	}
}

func TestQueueRetentionIncidentDeletionsSelectsClosedIncidentsOnly(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t, ctx)
	openIncident, err := repo.CreateIncident(ctx, "open", "")
	if err != nil {
		t.Fatalf("create open incident: %v", err)
	}
	closedIncident, err := repo.CreateIncident(ctx, "closed", "")
	if err != nil {
		t.Fatalf("create closed incident: %v", err)
	}
	if _, err := repo.CloseIncident(ctx, closedIncident.ID); err != nil {
		t.Fatalf("close incident: %v", err)
	}

	queued, err := repo.QueueRetentionIncidentDeletions(ctx, time.Now().UTC().Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("queue retention deletions: %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued retention deletions = %d, want 1", queued)
	}
	status, err := repo.GetIncidentDeletionStatus(ctx, closedIncident.ID)
	if err != nil {
		t.Fatalf("get closed incident deletion: %v", err)
	}
	if status.Source != incidents.IncidentDeletionSourceRetentionPolicy || status.ReasonCode != "closed_incident_retention" {
		t.Fatalf("unexpected retention deletion status: %+v", status)
	}
	if _, err := repo.GetIncidentDeletionStatus(ctx, openIncident.ID); !errors.Is(err, incidents.ErrNotFound) {
		t.Fatalf("open incident deletion status error = %v, want ErrNotFound", err)
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
