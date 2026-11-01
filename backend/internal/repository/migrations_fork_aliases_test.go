package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"regexp"
	"testing"
	"testing/fstest"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

const (
	forkAliasOld = "242_drop_platform_check_constraints.sql"
	forkAliasNew = "249_drop_platform_check_constraints.sql"
	forkAliasSQL = "CREATE TABLE fork_migration_alias_test(id int);"
)

func expectForkAliasUnlock(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
		WithArgs(migrationsAdvisoryLockID).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestForkV0219MigrationAliasesKeepSQLUnchanged(t *testing.T) {
	require.Len(t, forkV0219MigrationAliases, 8)
	hashes := map[string]string{
		"249_drop_platform_check_constraints.sql":       "89b9ca37806f5725fd01cd3b45d006c119d34820e29e1b8a90c0a8ba461db5bb",
		"250_group_i18n.sql":                            "9df295380bd4349bb510a64a3f3d829d7ee28904eab2bdf60ec50599b2128f28",
		"251_gemini_cached_contents.sql":                "48d05b5795e22f45458fdd56e32e3af89ebc68002ddb9320bff4d19bacc71dc3",
		"252_gemini_cached_contents_channel_usage.sql":  "282f55ed98d623f0a07cf4e7975f3ba0de09777ec664de612817a0ed2698577d",
		"253_gemini_cached_contents_mutation_lease.sql": "d9e2fdc9e974cf35870cf0d46dba34399fde760a08e97632d6d20abae412ea25",
		"254_enable_gemini_image_generation_groups.sql": "f49b458cfbd52bd17f5bb02e30e3813c8e192e4ae38bce6358713ef65f3874b5",
		"255_scoped_admin_api_keys.sql":                 "0c4c22f564f276fb305a54cf2d6c4b1a6179d748d5d8b74a3b99676282355bf6",
		"256_group_image_count_billing.sql":             "c91d6d31937658e71c68fa3af6a26fc47380f61243a681ac275aeb4706b242ab",
	}
	for name, oldName := range forkV0219MigrationAliases {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, oldName, name)
			content, err := migrations.FS.ReadFile(name)
			require.NoError(t, err)
			sum := sha256.Sum256(content)
			require.Equal(t, hashes[name], hex.EncodeToString(sum[:]))
		})
	}
}

func TestApplyMigrationsFS_ForkAliasSkipsPreviouslyAppliedSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	prepareMigrationsBootstrapExpectations(mock)
	checksum := migrationChecksum(forkAliasSQL)
	selectChecksum := regexp.QuoteMeta("SELECT checksum FROM schema_migrations WHERE filename = $1")

	mock.ExpectQuery(selectChecksum).WithArgs(forkAliasOld).
		WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow(checksum))
	mock.ExpectExec("INSERT INTO schema_migrations \\(filename, checksum, applied_at\\)").
		WithArgs(forkAliasNew, forkAliasOld, checksum).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(selectChecksum).WithArgs(forkAliasNew).
		WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow(checksum))
	expectForkAliasUnlock(mock)

	fsys := fstest.MapFS{forkAliasNew: &fstest.MapFile{Data: []byte(forkAliasSQL)}}
	require.NoError(t, applyMigrationsFS(context.Background(), db, fsys))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyMigrationsFS_ForkAliasFreshDatabaseRunsSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	prepareMigrationsBootstrapExpectations(mock)
	checksum := migrationChecksum(forkAliasSQL)
	selectChecksum := regexp.QuoteMeta("SELECT checksum FROM schema_migrations WHERE filename = $1")

	mock.ExpectQuery(selectChecksum).WithArgs(forkAliasOld).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(selectChecksum).WithArgs(forkAliasNew).WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(forkAliasSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)")).
		WithArgs(forkAliasNew, checksum).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectForkAliasUnlock(mock)

	fsys := fstest.MapFS{forkAliasNew: &fstest.MapFile{Data: []byte(forkAliasSQL)}}
	require.NoError(t, applyMigrationsFS(context.Background(), db, fsys))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyMigrationsFS_ForkAliasRejectsChecksumMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	prepareMigrationsBootstrapExpectations(mock)
	selectChecksum := regexp.QuoteMeta("SELECT checksum FROM schema_migrations WHERE filename = $1")

	mock.ExpectQuery(selectChecksum).WithArgs(forkAliasOld).
		WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow("different-checksum"))
	expectForkAliasUnlock(mock)

	fsys := fstest.MapFS{forkAliasNew: &fstest.MapFile{Data: []byte(forkAliasSQL)}}
	err = applyMigrationsFS(context.Background(), db, fsys)
	require.ErrorContains(t, err, "prior migration alias")
	require.ErrorContains(t, err, "checksum mismatch")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdoptForkMigrationAlias_ErrorPaths(t *testing.T) {
	const expected = "test-checksum"
	for _, tc := range []struct {
		name  string
		query error
		write error
	}{
		{"query_error", errors.New("DB unavailable"), nil},
		{"record_error", nil, errors.New("write failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			selectChecksum := regexp.QuoteMeta("SELECT checksum FROM schema_migrations WHERE filename = $1")

			if tc.query != nil {
				mock.ExpectQuery(selectChecksum).WithArgs(forkAliasOld).WillReturnError(tc.query)
			} else {
				mock.ExpectQuery(selectChecksum).WithArgs(forkAliasOld).
					WillReturnRows(sqlmock.NewRows([]string{"checksum"}).AddRow(expected))
				mock.ExpectExec("INSERT INTO schema_migrations \\(filename, checksum, applied_at\\)").
					WithArgs(forkAliasNew, forkAliasOld, expected).WillReturnError(tc.write)
			}
			require.Error(t, adoptForkMigrationAlias(context.Background(), db, forkAliasNew, expected))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAdoptForkMigrationAlias_IgnoresUnmappedMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	require.NoError(t, adoptForkMigrationAlias(context.Background(), db, "001_init.sql", "anything"))
	require.NoError(t, mock.ExpectationsWereMet())
}
