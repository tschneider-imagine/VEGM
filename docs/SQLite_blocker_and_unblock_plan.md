# SQLite blocker and unblock plan for VEGM

Date: 2026-05-20

## What the blocker actually is

The blocker is not that SQLite is a bad fit for VEGM.
The blocker is that the current build environment used during this project work could not cleanly add and exercise a Go SQLite driver.

## What was directly confirmed
- Go was present
- `CGO_ENABLED=1`
- `gcc` was present
- `sqlite3` CLI was not installed
- no SQLite Go driver was already vendored or cached
- fetching a SQLite Go driver from the Go module proxy failed in that environment

## Practical conclusion
SQLite remains a good fit for VEGM as a **search and indexing layer**.
The blocker is dependency acquisition and runtime validation, not the design choice itself.

## Recommended architecture
Keep:
- JSONL event log as the current source of truth
- raw payload capture on disk

Add later:
- SQLite as a local searchable index over event metadata, state transitions, and run metadata

Do not make SQLite the only storage path for raw payloads in the first pass.

## Preferred unblock path
### Recommended
Vendor a pure-Go SQLite driver such as `modernc.org/sqlite` into the repo.

Why this is the preferred fit:
- lower operational friction for a lightweight multi-instance VEGM fleet
- cleaner offline builds after vendoring
- no extra CGO burden as the primary plan

### Fallback
Vendor `mattn/go-sqlite3` and accept CGO in the build chain.

This is still workable, but it is a less elegant default for the low-overhead VEGM goal.

## Exact next external step
On a connected developer machine:
1. choose the SQLite driver
2. add it to the VEGM Go module
3. run `go mod tidy`
4. run `go mod vendor`
5. commit or zip the updated source tree with `vendor/`

Once that vendored dependency is in the repo, SQLite integration can be completed offline.

## Planned SQLite use once unblocked
1. startup migration/bootstrap
2. event metadata index
3. log query acceleration
4. run metadata tables
5. optional correlation tables for payload files and state transitions

## Important boundary
Until the driver is vendored, continue advancing VEGM with:
- JSONL logs
- payload capture
- query interfaces that can later be backed by SQLite without breaking callers
