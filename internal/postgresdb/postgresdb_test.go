package postgresdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-proofline/server/internal/auth"
	"github.com/open-proofline/server/internal/incidents"
	"golang.org/x/crypto/bcrypt"
)

func TestPostgresMigrateCreatesSchemaAndRejectsChecksumMismatch(t *testing.T) {
	ctx := context.Background()
	conn := openPostgresTestDB(t, ctx)

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	assertPostgresTable(t, ctx, conn, "schema_migrations")
	assertPostgresTable(t, ctx, conn, "incidents")
	assertPostgresTable(t, ctx, conn, "media_streams")
	assertPostgresTable(t, ctx, conn, "chunks")
	assertPostgresTable(t, ctx, conn, "checkins")
	assertPostgresTable(t, ctx, conn, "incident_tokens")
	assertPostgresTable(t, ctx, conn, "accounts")
	assertPostgresTable(t, ctx, conn, "auth_sessions")

	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `
		UPDATE schema_migrations
		SET checksum = 'bad'
		WHERE id = '001_init.sql'`); err != nil {
		t.Fatalf("update checksum: %v", err)
	}
	err := Migrate(ctx, conn)
	if err == nil {
		t.Fatal("expected checksum mismatch")
	}
	if !strings.Contains(err.Error(), "001_init.sql checksum mismatch") {
		t.Fatalf("expected checksum mismatch for 001_init.sql, got %v", err)
	}
}

func TestPostgresMigrateSerializesConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	conns := openPostgresTestDBsInOneSchema(t, ctx, 4)

	start := make(chan struct{})
	errCh := make(chan error, len(conns))
	var wg sync.WaitGroup
	for _, conn := range conns {
		conn := conn
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- Migrate(ctx, conn)
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent Migrate: %v", err)
		}
	}
	assertPostgresTable(t, ctx, conns[0], "schema_migrations")
	assertPostgresTable(t, ctx, conns[0], "incidents")

	var records int
	if err := conns[0].QueryRowContext(ctx, `
		SELECT count(*)
		FROM schema_migrations
		WHERE id = '001_init.sql'`,
	).Scan(&records); err != nil {
		t.Fatalf("count migration records: %v", err)
	}
	if records != 1 {
		t.Fatalf("migration record count = %d, want 1", records)
	}
}

func TestPostgresSchemaConstraints(t *testing.T) {
	ctx := context.Background()
	conn := openPostgresTestDB(t, ctx)
	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Now().UTC()
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO incidents (id, created_at, updated_at, status)
		VALUES ($1, $2, $3, $4)`,
		"inc_valid",
		now,
		now,
		incidents.StatusOpen,
	); err != nil {
		t.Fatalf("insert valid incident: %v", err)
	}
	assertPostgresConstraint(t, execErr(conn.ExecContext(ctx, `
		INSERT INTO incidents (id, created_at, updated_at, status)
		VALUES ($1, $2, $3, $4)`,
		"inc_bad_status",
		now,
		now,
		"paused",
	)))
	assertPostgresConstraint(t, execErr(conn.ExecContext(ctx, `
		INSERT INTO media_streams (id, incident_id, media_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		"str_bad_media",
		"inc_valid",
		"image",
		incidents.StreamStatusOpen,
		now,
		now,
	)))
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO media_streams (id, incident_id, media_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		"str_audio",
		"inc_valid",
		incidents.MediaTypeAudio,
		incidents.StreamStatusOpen,
		now,
		now,
	); err != nil {
		t.Fatalf("insert valid stream: %v", err)
	}
	assertPostgresConstraint(t, execErr(conn.ExecContext(ctx, `
		INSERT INTO chunks (
			id, incident_id, stream_id, chunk_index, media_type, started_at, ended_at,
			stored_path, byte_size, sha256_hex, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		"chk_bad_index",
		"inc_valid",
		"str_audio",
		0,
		incidents.MediaTypeAudio,
		now,
		now,
		"incidents/inc_valid/streams/str_audio/audio_000000.enc",
		int64(1),
		strings.Repeat("a", 64),
		now,
	)))
	assertPostgresConstraint(t, execErr(conn.ExecContext(ctx, `
		INSERT INTO chunks (
			id, incident_id, stream_id, chunk_index, media_type, started_at, ended_at,
			stored_path, byte_size, sha256_hex, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		"chk_bad_sha",
		"inc_valid",
		"str_audio",
		1,
		incidents.MediaTypeAudio,
		now,
		now,
		"incidents/inc_valid/streams/str_audio/audio_000001.enc",
		int64(1),
		"not-a-sha",
		now,
	)))
	assertPostgresConstraint(t, execErr(conn.ExecContext(ctx, `
		INSERT INTO chunks (
			id, incident_id, stream_id, chunk_index, media_type, started_at, ended_at,
			stored_path, byte_size, sha256_hex, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		"chk_wrong_stream_media",
		"inc_valid",
		"str_audio",
		1,
		incidents.MediaTypeVideo,
		now,
		now,
		"incidents/inc_valid/streams/str_audio/video_000001.enc",
		int64(1),
		strings.Repeat("a", 64),
		now,
	)))
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO incident_tokens (id, incident_id, token_hash, created_at)
		VALUES ($1, $2, $3, $4)`,
		"itk_valid",
		"inc_valid",
		strings.Repeat("b", 64),
		now,
	); err != nil {
		t.Fatalf("insert valid token: %v", err)
	}
	assertPostgresConstraint(t, execErr(conn.ExecContext(ctx, `
		INSERT INTO incident_tokens (id, incident_id, token_hash, created_at)
		VALUES ($1, $2, $3, $4)`,
		"itk_duplicate",
		"inc_valid",
		strings.Repeat("b", 64),
		now,
	)))
}

func TestPostgresRepositoryPreservesCoreSemantics(t *testing.T) {
	ctx := context.Background()
	conn := openPostgresTestDB(t, ctx)
	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	repo := NewRepository(conn)

	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	firstStream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "first audio")
	if err != nil {
		t.Fatalf("create first stream: %v", err)
	}
	secondStream, err := repo.CreateMediaStream(ctx, incident.ID, incidents.MediaTypeAudio, "second audio")
	if err != nil {
		t.Fatalf("create second stream: %v", err)
	}

	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, firstStream.ID, incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create first stream chunk: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, secondStream.ID, incidents.MediaTypeAudio, 1)); err != nil {
		t.Fatalf("create second stream chunk: %v", err)
	}
	legacy, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, "", incidents.MediaTypeAudio, 1))
	if err != nil {
		t.Fatalf("create legacy chunk: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, firstStream.ID, incidents.MediaTypeAudio, 1)); !errors.Is(err, incidents.ErrDuplicate) {
		t.Fatalf("duplicate stream chunk error = %v, want ErrDuplicate", err)
	}
	got, err := repo.GetChunkByKey(ctx, incident.ID, incidents.MediaTypeAudio, 1)
	if err != nil {
		t.Fatalf("get legacy chunk: %v", err)
	}
	if got.ID != legacy.ID || got.StreamID != "" {
		t.Fatalf("expected legacy chunk %+v, got %+v", legacy, got)
	}
	if _, err := repo.CompleteMediaStream(ctx, incident.ID, firstStream.ID, 1); err != nil {
		t.Fatalf("complete stream: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, firstStream.ID, incidents.MediaTypeAudio, 2)); !errors.Is(err, incidents.ErrInvalidState) {
		t.Fatalf("create chunk on completed stream error = %v, want ErrInvalidState", err)
	}
	if _, err := repo.CloseIncident(ctx, incident.ID); err != nil {
		t.Fatalf("close incident: %v", err)
	}
	if _, err := repo.CreateChunk(ctx, testChunkParams(incident.ID, "", incidents.MediaTypeAudio, 2)); !errors.Is(err, incidents.ErrIncidentClosed) {
		t.Fatalf("create chunk on closed incident error = %v, want ErrIncidentClosed", err)
	}
}

func TestPostgresRepositoryHashesAndRevokesIncidentTokens(t *testing.T) {
	ctx := context.Background()
	conn := openPostgresTestDB(t, ctx)
	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	repo := NewRepository(conn)
	incident, err := repo.CreateIncident(ctx, "phone", "")
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	token, rawToken, err := repo.CreateIncidentToken(ctx, incident.ID, "trusted contact", nil)
	if err != nil {
		t.Fatalf("create incident token: %v", err)
	}
	if rawToken == "" {
		t.Fatal("raw token was empty")
	}
	var storedHash string
	if err := conn.QueryRowContext(ctx, `
		SELECT token_hash
		FROM incident_tokens
		WHERE id = $1`,
		token.ID,
	).Scan(&storedHash); err != nil {
		t.Fatalf("read token hash: %v", err)
	}
	if storedHash == rawToken || len(storedHash) != 64 {
		t.Fatalf("token storage did not use a 64-character hash")
	}
	lookedUp, err := repo.LookupIncidentToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("lookup token: %v", err)
	}
	if lookedUp.ID != token.ID {
		t.Fatalf("looked up token id = %q, want %q", lookedUp.ID, token.ID)
	}
	if err := repo.RevokeIncidentToken(ctx, token.ID); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if _, err := repo.LookupIncidentToken(ctx, rawToken); !errors.Is(err, incidents.ErrNotFound) {
		t.Fatalf("lookup revoked token error = %v, want ErrNotFound", err)
	}
}

func TestPostgresRepositoryAccountsAndSessions(t *testing.T) {
	ctx := context.Background()
	conn := openPostgresTestDB(t, ctx)
	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	repo := NewRepository(conn)

	hasAccounts, err := repo.HasAccounts(ctx)
	if err != nil {
		t.Fatalf("has accounts: %v", err)
	}
	if hasAccounts {
		t.Fatal("expected fresh schema to have no accounts")
	}
	hasAdmin, err := repo.HasAdminAccount(ctx)
	if err != nil {
		t.Fatalf("has admin: %v", err)
	}
	if hasAdmin {
		t.Fatal("expected fresh schema to have no admin accounts")
	}

	adminPasswordHash, err := auth.HashPassword("test-password", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash admin password: %v", err)
	}
	admin, err := repo.CreateAccount(ctx, auth.CreateAccountParams{
		Username:     "Admin.User",
		PasswordHash: adminPasswordHash,
		Role:         auth.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create admin account: %v", err)
	}
	if admin.Username != "admin.user" || admin.Role != auth.RoleAdmin {
		t.Fatalf("unexpected admin account: %+v", admin)
	}
	if _, err := repo.CreateAccount(ctx, auth.CreateAccountParams{
		Username:     "admin.user",
		PasswordHash: adminPasswordHash,
		Role:         auth.RoleAdmin,
	}); !errors.Is(err, auth.ErrDuplicate) {
		t.Fatalf("duplicate account error = %v, want ErrDuplicate", err)
	}

	hasAccounts, err = repo.HasAccounts(ctx)
	if err != nil {
		t.Fatalf("has accounts after create: %v", err)
	}
	if !hasAccounts {
		t.Fatal("expected account existence after create")
	}
	hasAdmin, err = repo.HasAdminAccount(ctx)
	if err != nil {
		t.Fatalf("has admin after create: %v", err)
	}
	if !hasAdmin {
		t.Fatal("expected admin existence after create")
	}

	gotAdmin, err := repo.GetAccountByUsername(ctx, " ADMIN.USER ")
	if err != nil {
		t.Fatalf("get admin by username: %v", err)
	}
	if gotAdmin.ID != admin.ID || gotAdmin.PasswordHash != adminPasswordHash {
		t.Fatalf("got admin %+v, want id %q and stored hash", gotAdmin, admin.ID)
	}

	updatedHash, err := auth.HashPassword("updated-password", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash updated password: %v", err)
	}
	updatedAdmin, err := repo.UpdateAccountPassword(ctx, admin.ID, updatedHash)
	if err != nil {
		t.Fatalf("update admin password: %v", err)
	}
	if updatedAdmin.PasswordHash != updatedHash {
		t.Fatal("updated account did not return new password hash")
	}
	if updatedAdmin.PasswordChangedAt.Before(admin.PasswordChangedAt) {
		t.Fatalf("password_changed_at moved backward: before=%s after=%s", admin.PasswordChangedAt, updatedAdmin.PasswordChangedAt)
	}

	userPasswordHash, err := auth.HashPassword("regular-password", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash user password: %v", err)
	}
	user, err := repo.CreateAccount(ctx, auth.CreateAccountParams{
		Username:     "regular-user",
		PasswordHash: userPasswordHash,
		Role:         auth.RoleUser,
	})
	if err != nil {
		t.Fatalf("create user account: %v", err)
	}
	incident, err := repo.CreateIncidentForAccount(ctx, user.ID, "phone", "")
	if err != nil {
		t.Fatalf("create owned incident: %v", err)
	}
	if incident.OwnerAccountID != user.ID {
		t.Fatalf("incident owner = %q, want %q", incident.OwnerAccountID, user.ID)
	}

	session, rawToken, err := repo.CreateSession(ctx, user.ID, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if rawToken == "" {
		t.Fatal("raw session token was empty")
	}
	var storedHash string
	if err := conn.QueryRowContext(ctx, `
		SELECT token_hash
		FROM auth_sessions
		WHERE id = $1`,
		session.ID,
	).Scan(&storedHash); err != nil {
		t.Fatalf("read session token hash: %v", err)
	}
	if storedHash == rawToken || len(storedHash) != 64 {
		t.Fatalf("session storage did not use a 64-character hash")
	}
	lookedUp, err := repo.LookupSession(ctx, rawToken)
	if err != nil {
		t.Fatalf("lookup session: %v", err)
	}
	if lookedUp.ID != session.ID || lookedUp.AccountID != user.ID {
		t.Fatalf("looked up session %+v, want session %q for account %q", lookedUp, session.ID, user.ID)
	}
	if err := repo.RevokeSession(ctx, session.ID); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if _, err := repo.LookupSession(ctx, rawToken); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("lookup revoked session error = %v, want ErrNotFound", err)
	}

	expiredSession, expiredRawToken, err := repo.CreateSession(ctx, user.ID, time.Now().UTC().Add(-time.Second))
	if err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	if _, err := repo.LookupSession(ctx, expiredRawToken); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("lookup expired session %q error = %v, want ErrNotFound", expiredSession.ID, err)
	}

	keptSession, keptRawToken, err := repo.CreateSession(ctx, user.ID, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatalf("create kept session: %v", err)
	}
	revokedSession, revokedRawToken, err := repo.CreateSession(ctx, user.ID, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatalf("create revokable session: %v", err)
	}
	revoked, err := repo.RevokeAccountSessions(ctx, user.ID, keptSession.ID)
	if err != nil {
		t.Fatalf("revoke account sessions: %v", err)
	}
	if revoked != 2 {
		t.Fatalf("revoked sessions = %d, want 2", revoked)
	}
	if _, err := repo.LookupSession(ctx, keptRawToken); err != nil {
		t.Fatalf("kept session lookup after account revoke: %v", err)
	}
	if _, err := repo.LookupSession(ctx, revokedRawToken); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("lookup revoked account session %q error = %v, want ErrNotFound", revokedSession.ID, err)
	}
}

func openPostgresTestDB(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()
	conn := openPostgresConnection(t, ctx, postgresTestDSN(t))
	schema := "proofline_test_" + randomHex(t, 8)
	quotedSchema := quotePostgresIdentifier(schema)
	if _, err := conn.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		_ = conn.Close()
		t.Fatalf("create test schema: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "SET search_path TO "+quotedSchema); err != nil {
		_ = conn.Close()
		t.Fatalf("set test schema search path: %v", err)
	}
	t.Cleanup(func() {
		_, _ = conn.ExecContext(context.Background(), "DROP SCHEMA "+quotedSchema+" CASCADE")
		_ = conn.Close()
	})
	return conn
}

func openPostgresTestDBsInOneSchema(t *testing.T, ctx context.Context, count int) []*sql.DB {
	t.Helper()
	if count <= 0 {
		t.Fatal("postgres test database count must be positive")
	}

	dsn := postgresTestDSN(t)
	admin := openPostgresConnection(t, ctx, dsn)
	schema := "proofline_test_" + randomHex(t, 8)
	quotedSchema := quotePostgresIdentifier(schema)
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		_ = admin.Close()
		t.Fatalf("create shared test schema: %v", err)
	}

	conns := make([]*sql.DB, 0, count)
	t.Cleanup(func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
		_, _ = admin.ExecContext(context.Background(), "DROP SCHEMA "+quotedSchema+" CASCADE")
		_ = admin.Close()
	})

	for range count {
		conn := openPostgresConnection(t, ctx, dsn)
		if _, err := conn.ExecContext(ctx, "SET search_path TO "+quotedSchema); err != nil {
			_ = conn.Close()
			t.Fatalf("set shared test schema search path: %v", err)
		}
		conns = append(conns, conn)
	}
	return conns
}

func postgresTestDSN(t *testing.T) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("SAFE_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("SAFE_POSTGRES_TEST_DSN is not set")
	}
	return dsn
}

func openPostgresConnection(t *testing.T, ctx context.Context, dsn string) *sql.DB {
	t.Helper()
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("open postgres test database failed")
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		t.Fatal("connect postgres test database failed; verify SAFE_POSTGRES_TEST_DSN")
	}
	return conn
}

func quotePostgresIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func randomHex(t *testing.T, byteCount int) string {
	t.Helper()
	buf := make([]byte, byteCount)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate random suffix: %v", err)
	}
	return hex.EncodeToString(buf)
}

func assertPostgresTable(t *testing.T, ctx context.Context, conn *sql.DB, tableName string) {
	t.Helper()
	var exists bool
	if err := conn.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", tableName).Scan(&exists); err != nil {
		t.Fatalf("check table %s: %v", tableName, err)
	}
	if !exists {
		t.Fatalf("expected table %s to exist", tableName)
	}
}

func execErr(_ sql.Result, err error) error {
	return err
}

func assertPostgresConstraint(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected postgres constraint error")
	}
	if !isIntegrityConstraint(err) {
		t.Fatalf("expected postgres integrity constraint error, got %v", err)
	}
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
