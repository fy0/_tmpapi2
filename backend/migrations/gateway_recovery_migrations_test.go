package migrations

import (
	"strings"
	"testing"
)

func TestGatewayRecoveryMigrationsAreAdditiveAndOrdered(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	index := map[string]int{}
	for i, e := range entries {
		index[e.Name()] = i
	}
	names := []string{"256_group_image_count_billing.sql", "257_error_passthrough_rule_action.sql", "258_gateway_conversation_observation.sql", "259_gateway_conversation_indexes_notx.sql"}
	previous := -1
	for _, name := range names {
		position, exists := index[name]
		if !exists || position <= previous {
			t.Fatalf("migration missing or out of order: %s", name)
		}
		previous = position
	}
	action, err := FS.ReadFile(names[1])
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"ADD COLUMN IF NOT EXISTS action", "WHEN passthrough_code AND passthrough_body THEN 'passthrough'", "ELSE 'rewrite'", "'retry'", "IF NOT EXISTS"} {
		if !strings.Contains(string(action), text) {
			t.Fatalf("missing compatibility guard %q", text)
		}
	}
	observation, err := FS.ReadFile(names[2])
	if err != nil {
		t.Fatal(err)
	}
	sql := string(observation)
	if strings.Contains(sql, "UPDATE ") || strings.Contains(sql, "NOT NULL DEFAULT") {
		t.Fatal("unknown historical conversation IDs must stay NULL")
	}
	for _, table := range []string{"usage_logs", "ops_error_logs"} {
		if !strings.Contains(sql, "ALTER TABLE "+table) {
			t.Fatal("missing table " + table)
		}
	}
	indexes, err := FS.ReadFile(names[3])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(indexes), "CREATE INDEX CONCURRENTLY IF NOT EXISTS") != 2 || strings.Count(string(indexes), "conversation_id, user_id, api_key_id, created_at DESC") != 2 {
		t.Fatal("expected scoped nontransactional correlation indexes")
	}
}
