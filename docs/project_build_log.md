# G2S Project Build Log

## v13 addendum — SQLite blocker clarified

### What was confirmed
- The current environment has Go 1.23.2 on linux/amd64.
- `CGO_ENABLED=1`.
- `gcc` is present.
- `sqlite3` CLI is not installed.
- No SQLite Go module was already cached or vendored.
- Attempting to fetch a SQLite Go driver failed because the environment could not reach the Go module proxy.

### Practical conclusion
The blocker is dependency acquisition and runtime validation in this environment, not SQLite as a design choice for VEGM.

### Recommended path
- Keep JSONL plus raw payload capture as the current source of truth.
- Add SQLite later as a search and indexing layer.
- Preferred driver path: vendor `modernc.org/sqlite` into the VEGM repo.
- Fallback driver path: vendor `mattn/go-sqlite3` and accept CGO in the build chain.

### Next required external step
Have a connected machine vendor the chosen SQLite driver into the VEGM source tree, then return that updated source tree so SQLite integration can be completed offline.
