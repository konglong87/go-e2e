# Coding file memory — implementation and acceptance ledger

## Scope and architecture

Base: `deec0c9`, clean `main`, pulled before editing. No pre-existing dirty files.
Only coding-mode Markdown project memory; no tenant/database changes, new branch,
worktree, vector service, or additional runtime dependency. Preserve MEMORY.md,
rooted file reads, existing budgets and Read fallback. Common path enforcement
owns write containment; memory owns bounded lexical ranking; query owns context.

## Ordered delivery queue

| Priority | Slice | Dependency | Acceptance | Status |
|---|---|---|---|---|
| P0 | Canonicalize missing writable roots safely | None | Real Write, fresh/existing symlink roots, escape denial, tool regressions/race | Committed/pushed `0c9c877` |
| P1 | Chinese short-query and English-boundary retrieval; weighted index metadata | P0 | 25 multilingual cases; ranking, invalid metadata, scope/budget/security regressions | Committed/pushed `c0154b8` |
| P1 | File write/index/cold-process and Read-fallback integration | Retrieval | Actual file contents and captured request; no tenant dependency | Verified; test-double transport only |
| P1 | Real-model lifecycle E2E | Deterministic gates, isolated configured provider | Save, cold recall, paraphrase Read, update, forget, readonly, isolation; inspect disk/request/response independently | Pending |
| P1 | Actual coding desktop entry acceptance | Current built app and isolated workspace | Native clicks and screenshots, new session recall; do not substitute browser/mocked model evidence | Pending |
| P2 | Targeted corrective iteration | Any demonstrated lifecycle failure | Reproduce first, minimal fix, rerun affected and negative cases | Conditional |

## Verification policy

Run offline tests before paid model calls. Keep credentials in owner-only persistent
validation directories outside the checkout, never /tmp or Git. Raw provider data
is local-only; retain requested screenshots under ignored
`desktop-v2/build/validation/20261022/`. Preserve global settings. Clean temporary
build/test data after acceptance. Native scripted acceptance and autonomous model
behavior are distinct gates. Framework leaderboard claims are not our acceptance.

## Change ledger

- Intended: `internal/tools/path.go`, focused path tests; focused memory retrieval
  module/tests; query file-memory integration tests; existing memory acceptance
  runner/tests; architecture documentation; this ledger.
- Completed: missing writable-root canonicalization; actual Write regression and escaping-child denial.
- Tests: reproduced first-write failure before fix; `go test -count=1 ./internal/tools/...` and focused `go test -race` pass (Go 1.26.6, offline dependencies).
- Commits / pushes: path slice is this commit; push immediately after committing. Each verified slice goes to origin/main.
- Explicit deferrals: embeddings/vector database/background consolidation unless
  controlled evidence establishes necessity; unrelated tenant issues; previously
  user-deferred physical second display and formal signing/distribution.

## Retrieval slice verification

The old implementation failed 12/25 lexical cases plus field-ranking coverage.
The new implementation passes the fixed corpus without claiming overall model
accuracy. Query-side integration verifies actual file writes through a fresh
symlinked config root, cold-process request context, Read fallback and readonly
bytes. Full `go test -count=1 ./...`, memory and focused query race tests, and
`go vet ./internal/memory ./internal/tools ./internal/query` pass. No new
library/model/storage dependency; body/index caps and memory budgets are unchanged.
Next: extend and run real-provider lifecycle acceptance, then build and inspect
the actual native coding interface. No native or real-model completion claimed yet.

## Real-provider harness slice

The runner now starts without a precreated memory directory, optionally uses a
symlinked config root, and adds Read fallback, activity readonly, update/cold recall,
forget/cold recall, Chinese short-query and English-word-boundary scenarios.
Fallback requires the actual Read output to contain the answer while the first
request lacks it; forgetting scans unindexed Markdown too. Six offline harness
unit tests pass. Copied provider credentials are deleted in finally. Real-provider
execution remains pending; no claim is based on harness tests alone.
