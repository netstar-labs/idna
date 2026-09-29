# Audit findings — netstar-labs/idna

**Pass**: A1 adversarial-audit (four report-only auditors + adversarial verify pass)
plus a least-code pass, run 2026-09-29. Scope: the owned root code (`idna.go`,
`doc.go`, `idna_test.go`, docs) — `internal/x/` is a deliberately vendored,
pinned, byte-faithful-on-purpose copy of `golang.org/x/net/idna` and
`golang.org/x/text`, frozen and out of scope for direct edits (read for root-cause
tracing only).

**Baseline** (`go vet`, `staticcheck`, `gofmt -l`): clean. `deadcode -test ./...`
flagged ~140 "unreachable func" results, but every one is inside
`internal/x/text/{transform,unicode/bidi,unicode/norm}` — vendored packages exporting
their full upstream API surface, most of which this repo's own code never calls
directly. Expected for a byte-faithful vendor copy; not a finding.

## Top line

Two real, adversarially-confirmed correctness bugs in the owned wrapper — both public
API behavior changes, filed as their own issues and fixed on this branch after
explicit scope confirmation. Four additional low-risk fixes applied in the same pass
(a duplicated constant, two doc-drift issues, and documentation of two
previously-undocumented edge behaviors). No simplification or dedup opportunities
cleared the bar for action.

## CONFIRMED — fixed on this branch (own commits, own issues)

| # | Finding | Issue | Status |
|---|---|---|---|
| C-1 | `ToASCII`/`ToASCIIErr` silently accepted invalid UTF-8 bytes (U+FFFD-substitute + encode, no error) — contradicted the documented "ok is false when malformed" contract. | #4 | **Fixed** — `utf8.ValidString(host)` guard added to `ToASCII`/`ToASCIIErr`/`ToUnicode`. Input-acceptance change: previously-accepted malformed input is now rejected. |
| C-2 | `ToUnicode` had no `allowUnderscore` param, so it couldn't round-trip a host `ToASCII(..., true)` (loose profile) just accepted — e.g. `_dmarc.example.com`. | #5 | **Fixed** — `ToUnicode(host string, allowUnderscore bool)`, mirroring `ToASCII`. Breaking signature change for every existing caller — flagged for human sign-off, not self-merged. |

Full reproductions, root-cause tracing, and the adversarial-skeptic verification
transcripts for both are in `audit-correctness.md`. Both survived an independent
skeptic explicitly tasked with refuting them before being fixed; neither could be
refuted. Both fixes carry their own regression tests, sabotage-verified (each test
confirmed to fail against the pre-fix code with an attributable message, then pass
restored).

## Applied in this pass (low-risk, no public-behavior change)

- **Reuse fix**: `idna.go` hand-maintained `const unicodeVersion = "15.0.0"`,
  duplicating the vendored, generated `xidna.UnicodeVersion`. `Unicode()` now returns
  `xidna.UnicodeVersion` directly; the hand-copy is deleted. Detail: `audit-simplify.md`.
- **Doc-drift fix**: a `grep` command documented as verification in three places
  (`internal/x/README.md`, `docs/architecture.md`, `docs/userguide.md`) matched its
  own instruction text when run unscoped; scoped to `--include='*.go'` in all three.
  Detail: `audit-docs.md`.
- **Doc-drift fix**: `internal/x/staticcheck.conf` called the vendor tree
  "verbatim"/"byte-identical," contradicting `internal/x/README.md`'s own accurate
  "pruned snapshot, not a verbatim mirror." Reworded to match. Detail: `audit-docs.md`.
- **Documentation-only**: two previously-undocumented MINOR gaps (no RFC 1035
  length enforcement; leading/empty-dot labels pass through) are now noted on
  `ToASCII`'s doc comment, without changing behavior — enabling the underlying
  vendored options (`VerifyDNSLength`, `RemoveLeadingDots`) would itself be a
  behavior change and is out of scope for this pass. Detail: `audit-correctness.md`.

Re-validation after applying: `go build`, `go vet`, `staticcheck`, `gofmt -l`, and
`go test ./... -race` all green.

## REFUTED / no action

- Auditors A and B both independently surfaced the same marginal
  `err→(string,bool)` collapse pattern in `ToASCII`/`ToUnicode` and both recommended
  against factoring it (near-zero net LOC savings for two call sites; see
  `audit-simplify.md` / `audit-dedup.md`). Cross-validated, not acted on.

## Dimension reports

- `audit-simplify.md` — auditor A + the least-code fix
- `audit-dedup.md` — auditor B
- `audit-correctness.md` — auditor C, with both adversarial-skeptic transcripts
- `audit-docs.md` — auditor D
