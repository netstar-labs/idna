# Audit — doc / comment vs code drift (auditor D)

Scope: README.md, doc.go, all `docs/*.md`, `internal/x/README.md`, `VENDOR`,
`internal/x/staticcheck.conf`, and every doc comment in `idna.go`.

## CONFIRMED — fixed in this pass

### 1. [MISLEADING] Self-defeating `grep` verification command

`internal/x/README.md`, `docs/architecture.md`, `docs/userguide.md` each instructed
verifying "no `//go:build go1.x` selector remains" with:

```
grep -r '//go:build go1' internal/x
```

Run literally, this returns **5 matches** — all inside `internal/x/README.md`'s own
prose, which quotes the string while explaining the stripping. The underlying
engineering claim is true (`grep -rn '//go:build' internal/x --include='*.go'` → 0
matches across all 24 vendored `.go` files, confirmed independently), but the
documented command as written is not the command that proves it, and it's given as
the literal verification step in the userguide's re-vendor procedure.

**Fix applied**: scoped all three occurrences to `--include='*.go'`, with a one-line
note on why (an unscoped grep also matches the sentence stating the command).

### 2. [COSMETIC] "Verbatim"/"byte-identical" contradicted `internal/x/README.md`

`internal/x/staticcheck.conf` described the vendor tree as "vendored verbatim... byte-
identical," while `internal/x/README.md` itself says: *"this is a pruned snapshot (not
a verbatim mirror)"* — accurate, since import paths are rewritten and pre-Unicode-15
files/build tags are deleted on vendor.

**Fix applied**: reworded `staticcheck.conf`'s comment to match `internal/x/README.md`'s
own, more accurate characterization; the actual point of the comment (don't lint
vendored code, don't fork it via local fixes) is unchanged.

## CONFIRMED CLEAN (independently verified, not assumed)

- Every code example in `README.md` and `docs/userguide.md` **executed** against the
  real package (not just read for plausibility) and matched exactly, including
  `ToASCII`/`ToASCIIErr`/`ToUnicode`/`Unicode()` outputs and the CJK/`faß.de`/`münchen.de`
  cases against `idna_test.go`'s own vectors.
- `unicodeVersion` claim of "15.0.0" cross-checked against three independent vendored
  table files (`idna/tables15.0.0.go`, `text/unicode/bidi/tables15.0.0.go`,
  `text/unicode/norm/tables15.0.0.go`) — all three independently declare 15.0.0 (not
  just checked against each other).
- `go.mod` module path matches every doc/README reference.
- Option semantics (`MapForLookup`, `Transitional`, `StrictDomainName`) verified
  against the actual vendored option constructors and their interaction order
  (later options override earlier ones) — matches `docs/architecture.md`'s table.
- `//go:build go1.x` removal claim (see #1 above) — the engineering claim itself is
  true, confirmed by a properly-scoped grep across all 24 vendored `.go` files.
- Vendor-pruning inventory (`internal/x/README.md`) — confirmed by directory listing:
  only the Unicode-15 table variant present per package, no `*_test.go` or generator
  files anywhere under `internal/x/`, upstream LICENSE/PATENTS preserved.
- `GOWORK=off go test ./...` — ran successfully; confirmed the instruction is
  operationally load-bearing (a `go.work` exists one directory up), not boilerplate.
- Every doc comment / signature pair in `idna.go` matches `docs/userguide.md`'s API
  table exactly.

## Unverifiable (reported, not independently confirmed)

- `VENDOR`'s pinned `golang.org/x/net v0.40.0` / `golang.org/x/text v0.25.0` against
  actual upstream source — those module versions were not present in the local module
  cache and no network fetch was attempted. Internal consistency (all docs agree with
  `VENDOR`) is confirmed; agreement with the real upstream release is not.
- Claims that `sanitize` and `normie` are the two real consumers, and the exact shape
  of normie's `Options.IDNA` seam — external to this repo, not checkable from within it.
