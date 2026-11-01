package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// forkV0219MigrationAliases maps the Fork-only sequence to the identical SQL
// shipped by the first (unshifted) v0.2.19 upgrade. Filenames, not just numeric
// prefixes, are identities in schema_migrations; keep these mappings immutable.
var forkV0219MigrationAliases = map[string]string{
	"249_drop_platform_check_constraints.sql":       "242_drop_platform_check_constraints.sql",
	"250_group_i18n.sql":                            "242_group_i18n.sql",
	"251_gemini_cached_contents.sql":                "243_gemini_cached_contents.sql",
	"252_gemini_cached_contents_channel_usage.sql":  "244_gemini_cached_contents_channel_usage.sql",
	"253_gemini_cached_contents_mutation_lease.sql": "245_gemini_cached_contents_mutation_lease.sql",
	"254_enable_gemini_image_generation_groups.sql": "246_enable_gemini_image_generation_groups.sql",
	"255_scoped_admin_api_keys.sql":                 "247_scoped_admin_api_keys.sql",
	"256_group_image_count_billing.sql":             "248_group_image_count_billing.sql",
}

// adoptForkMigrationAlias avoids executing already-applied SQL a second time
// on a DB that ran the first v0.2.19 binary with upstream names. Keep the
// original record as well, so older binaries still recognize applied SQL.
// The caller holds the migrations advisory lock. Unknown checksums fail closed.
func adoptForkMigrationAlias(ctx context.Context, db migrationConnection, name, checksum string) error {
	oldName, ok := forkV0219MigrationAliases[name]
	if !ok {
		return nil
	}

	var appliedChecksum string
	err := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", oldName).Scan(&appliedChecksum)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check prior migration alias %s: %w", oldName, err)
	}
	if appliedChecksum != checksum {
		return fmt.Errorf("prior migration alias %s checksum mismatch (db=%s file=%s)", oldName, appliedChecksum, checksum)
	}

	// Only duplicate the verified migration metadata; never repeat its SQL.
	// If the new name already exists, the main runner validates its checksum.
	_, err = db.ExecContext(ctx,
		"INSERT INTO schema_migrations (filename, checksum, applied_at) "+
			"SELECT $1, checksum, applied_at FROM schema_migrations "+
			"WHERE filename = $2 AND checksum = $3 "+
			"ON CONFLICT (filename) DO NOTHING",
		name, oldName, checksum)
	if err != nil {
		return fmt.Errorf("record migration alias %s -> %s: %w", oldName, name, err)
	}
	return nil
}
