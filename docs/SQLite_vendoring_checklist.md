# SQLite vendoring checklist

Use this when the team is ready to enable the VEGM SQLite-backed index layer.

## Goal

Provide the repo with an offline-usable SQLite Go dependency so the runtime can add a real index/search backend without depending on live network access.

## Recommended path

Preferred driver approach:
- vendor a pure-Go SQLite driver into this repository
- keep JSONL and payload capture as the runtime source of truth
- use SQLite first as a search/index layer

## Connected-machine steps

From the VEGM repository root on a machine with internet access:

```bash
go get modernc.org/sqlite@latest
go mod tidy
go mod vendor
```

Then commit or zip the updated repository including:
- `go.mod`
- `go.sum`
- `vendor/`

## What to do after vendoring

1. add a concrete SQLite implementation under `storage/`
2. keep the `storage.Index` interface intact
3. wire the runtime logger/search layer to mirror event metadata into the index
4. keep raw payload files on disk in the first pass
5. add migration/bootstrap tests

## Guardrails

- do not replace JSONL logging as the primary truth source immediately
- do not move raw payload bodies into SQLite first
- do not block VEGM runtime startup on optional index features until the implementation is proven
