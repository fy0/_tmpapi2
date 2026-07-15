//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var forkV0155MigrationNames = []string{
	"185_add_usage_log_long_context_billing.sql",
	"186_add_usage_logs_api_key_latest_ip_index_notx.sql",
	"187_group_web_search_price_per_call.sql",
	"188_add_ops_system_logs_host.sql",
	"189_default_openai_long_context_billing.sql",
	"190_add_ops_system_logs_host_index_notx.sql",
	"191_channel_monitor_grok_provider.sql",
}

func TestForkV0155MigrationsUpgradeExistingSchema(t *testing.T) {
	ctx := context.Background()
	db := openForkUpgradeTestDatabase(t)

	require.NoError(t, applyMigrationsFS(ctx, db, migrationsBeforeForkV0155(t)))

	var groupID int64
	require.NoError(t, db.QueryRowContext(ctx, `
INSERT INTO groups (
    name,
    platform,
    allow_image_generation,
    responses_image_generation_redirect_group_id,
    peak_rate_enabled,
    peak_start,
    peak_end,
    peak_rate_multiplier
)
VALUES ('fork-v0155-upgrade', 'openai', TRUE, 777, TRUE, '09:00', '10:00', 1.5)
RETURNING id
`).Scan(&groupID))

	var ordinaryID int64
	require.NoError(t, db.QueryRowContext(ctx, `
INSERT INTO accounts (name, platform, type, extra)
VALUES ('fork-v0155-ordinary', 'openai', 'oauth', '{}'::jsonb)
RETURNING id
`).Scan(&ordinaryID))

	var parentID int64
	require.NoError(t, db.QueryRowContext(ctx, `
INSERT INTO accounts (name, platform, type, extra)
VALUES ('fork-v0155-parent', 'openai', 'oauth', '{"openai_long_context_billing_enabled":true}'::jsonb)
RETURNING id
`).Scan(&parentID))

	var shadowID int64
	require.NoError(t, db.QueryRowContext(ctx, `
INSERT INTO accounts (name, platform, type, extra, parent_account_id, quota_dimension)
VALUES ('fork-v0155-shadow', 'openai', 'oauth', '{}'::jsonb, $1, 'spark')
RETURNING id
`, parentID).Scan(&shadowID))

	require.NoError(t, applyMigrationsFS(ctx, db, dbmigrations.FS))
	require.NoError(t, applyMigrationsFS(ctx, db, dbmigrations.FS), "upgraded migration set must remain idempotent")

	for _, name := range forkV0155MigrationNames {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", name).Scan(&count))
		require.Equal(t, 1, count, "migration %s should be recorded exactly once", name)
	}

	for _, obsolete := range []string{
		"174_add_usage_log_long_context_billing.sql",
		"174_add_usage_logs_api_key_latest_ip_index_notx.sql",
		"174_group_web_search_price_per_call.sql",
		"175_add_ops_system_logs_host.sql",
		"175_default_openai_long_context_billing.sql",
		"175a_add_ops_system_logs_host_index_notx.sql",
		"176_channel_monitor_grok_provider.sql",
	} {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", obsolete).Scan(&count))
		require.Zero(t, count, "obsolete upstream migration alias must not be recorded: %s", obsolete)
	}

	var (
		allowImages bool
		redirectID  sql.NullInt64
		peakEnabled bool
		peakRate    float64
		webPrice    sql.NullFloat64
	)
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT
    allow_image_generation,
    responses_image_generation_redirect_group_id,
    peak_rate_enabled,
    peak_rate_multiplier,
    web_search_price_per_call
FROM groups
WHERE id = $1
`, groupID).Scan(&allowImages, &redirectID, &peakEnabled, &peakRate, &webPrice))
	require.True(t, allowImages)
	require.Equal(t, int64(777), redirectID.Int64)
	require.True(t, peakEnabled)
	require.InDelta(t, 1.5, peakRate, 0.0001)
	require.False(t, webPrice.Valid)

	var ordinaryEnabled bool
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT (extra->>'openai_long_context_billing_enabled')::boolean FROM accounts WHERE id = $1
`, ordinaryID).Scan(&ordinaryEnabled))
	require.False(t, ordinaryEnabled)

	var shadowEnabled bool
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT (extra->>'openai_long_context_billing_enabled')::boolean FROM accounts WHERE id = $1
`, shadowID).Scan(&shadowEnabled))
	require.True(t, shadowEnabled)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	requireColumn(t, tx, "usage_logs", "long_context_billing_applied", "boolean", 0, false)
	requireColumn(t, tx, "groups", "web_search_price_per_call", "numeric", 0, true)
	requireColumn(t, tx, "ops_system_logs", "host", "character varying", 255, true)
	requireIndex(t, tx, "usage_logs", "idx_usage_logs_api_key_latest_ip")
	requireIndex(t, tx, "ops_system_logs", "idx_ops_system_logs_host_created_at")
	requireConstraintDefinitionContains(t, tx, "channel_monitors", "channel_monitors_provider_check", "'grok'")
	requireConstraintDefinitionContains(t, tx, "channel_monitor_request_templates", "channel_monitor_request_templates_provider_check", "'grok'")

	for _, table := range []string{"invoice_requests", "support_tickets", "batch_image_jobs"} {
		var regclass sql.NullString
		require.NoError(t, tx.QueryRowContext(ctx, "SELECT to_regclass($1)", "public."+table).Scan(&regclass))
		require.True(t, regclass.Valid, "fork table must survive upgrade: %s", table)
	}
}

func migrationsBeforeForkV0155(t *testing.T) fstest.MapFS {
	t.Helper()
	excluded := make(map[string]struct{}, len(forkV0155MigrationNames))
	for _, name := range forkV0155MigrationNames {
		excluded[name] = struct{}{}
	}

	entries, err := dbmigrations.FS.ReadDir(".")
	require.NoError(t, err)
	result := make(fstest.MapFS)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		if _, skip := excluded[name]; skip {
			continue
		}
		content, readErr := dbmigrations.FS.ReadFile(name)
		require.NoError(t, readErr)
		result[name] = &fstest.MapFile{Data: content, Mode: 0o444}
	}
	return result
}

func openForkUpgradeTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	databaseName := fmt.Sprintf("sub2api_upgrade_%d", time.Now().UnixNano())
	_, err := integrationDB.ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(databaseName))
	require.NoError(t, err)

	dsn, err := url.Parse(integrationDSN)
	require.NoError(t, err)
	dsn.Path = "/" + databaseName
	db, err := openSQLWithRetry(ctx, dsn.String(), 30*time.Second)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
		_, _ = integrationDB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+pq.QuoteIdentifier(databaseName)+" WITH (FORCE)")
	})
	return db
}
