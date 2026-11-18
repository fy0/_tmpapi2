# Fork migration sequence policy

## 2026-11-19: upstream v0.2.20 through v0.2.23

Compared all four upstream migration batches against v0.2.19 and the current
Fork. The target upstream high-water prefix is 252; the existing Fork
high-water prefix is 259. This batch maps each new SQL independently, in final
upstream lexical order:

| Upstream | Fork |
| --- | --- |
| 249_add_usage_logs_api_key_cost_index_notx.sql | 260_add_usage_logs_api_key_cost_index_notx.sql |
| 249_mcp_oauth.sql | 261_mcp_oauth.sql |
| 250_openai_cache_write_ttl.sql | 262_openai_cache_write_ttl.sql |
| 251_group_force_openai_ultrafast.sql | 263_group_force_openai_ultrafast.sql |
| 251_group_usage_rollup_nonblocking_build.sql | 264_group_usage_rollup_nonblocking_build.sql |
| 252_channel_pricing_ultrafast_multiplier.sql | 265_channel_pricing_ultrafast_multiplier.sql |

The cost index is introduced in v0.2.21, but sorts before the independent MCP
migration introduced in v0.2.20. Intermediate merge commits are not releases.
All 309 historical SQL files through 259 retain their filenames and bytes;
new files retain upstream SQL bytes. The runner's nontransactional index
recovery and all explicit references use the mapped filenames. The historical
v0.2.19 alias whitelist is unchanged; this batch adds no speculative aliases.
`fork_v0223_sequence_test.go` verifies the new order, upstream byte checksums,
and the aggregate checksum of all historical SQL.

260–265 is this batch's historical mapping, not the next release allocation
rule. Dependencies, raw and runner checksums, verification, and deployment
gates are documented in `UPGRADE_V0.2.23_2026-11-19.md`.

**Production deployment remains BLOCKED:** target history/structure checks,
backups, and isolated real PostgreSQL snapshot upgrade/rollback rehearsals
have not been performed. Pushing code for Actions to build does not authorize
deployment.

## 2026-11-10: gateway recovery / conversation observations

The checked-out upstream baseline is v0.2.19 (highest upstream prefix 248); the pre-feature Fork high-water mark is 256. This exact batch allocates:

- `257_error_passthrough_rule_action.sql`: additive action with legacy passthrough/rewrite backfill.
- `258_gateway_conversation_observation.sql`: nullable correlation/source/metadata on usage and error logs; historical unknown IDs stay NULL.
- `259_gateway_conversation_indexes_notx.sql`: scope-aware partial indexes, built concurrently outside a transaction. The runner only drops invalid indexes of these exact names when recovering an interrupted build.

This allocation does not fix the next batch's starting number. Recompute upstream/Fork boundaries and dependencies each time. Historical 001–256 migrations remain unchanged. `TestGatewayRecoveryMigrationsAreAdditiveAndOrdered` supplements the existing checksum/alias tests.

**Production deployment remains BLOCKED:** production history/table checks, backups, isolated PostgreSQL upgrade and rollback rehearsals have not been performed by this task.

## Permanent policy

1. A migration filename is a durable database identity, stored in schema_migrations with a SHA-256 checksum. Never edit, rename, reorder, or delete an already-applied Fork migration.
2. This Fork added seven business-specific SQL files (154–160) relative to upstream v0.2.13 and renumbered a subset of upstream migrations (161–212); later names through 241 are shared. Compare by filenames **and** SQL checksums, not numeric prefixes alone.
3. The upstream v0.2.15–v0.2.19 SQL files were reassigned to Fork-only numbers 249–256 (all original SQL bytes preserved). Upstream 242–248 are reserved and MUST NOT be reintroduced under their upstream names.
4. **Do not hard-code the next Fork migration number.** Before each upstream upgrade, diff the exact old/new upstream migration sets, compare against the current Fork files and deployed-history mapping, and inspect SQL dependencies. Compute the batch numbering from the **highest prefix actually present in the target upstream batch and the highest already allocated in this Fork**; choose safe unused identifiers strictly after both boundaries, preserving the new migrations execution order. Upstream may jump in numbering or use the same prefix more than once; the Fork must map each new SQL individually and may leave gaps. The historical 249–256 mapping is not a forecast for the next release. Update all filename references and regression tests. Do not treat matching names or arithmetic offsets as evidence of migration identity.
5. A database that ran the initial unshifted v0.2.19 upgrade may have schema_migrations entries with upstream 242–248 filenames. The migrations runner recognizes precisely these old-name aliases only when their content checksum matches; it adds a new-name metadata record without executing the SQL again, preserving the original applied_at and old entry.
6. The presence of old aliases must be checked before any production deployment. An unknown checksum is an error requiring manual investigation; never bypass checksum checks. Rehearse the migration on an isolated real PostgreSQL snapshot before deploying.
7. Preserve the source and meaning of historical migration names on subsequent upstream merges. Run TestForkV0219MigrationsFollowExistingSequence and full migration unit tests after each upgrade.
