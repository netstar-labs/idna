# Audit — duplication / dedup (auditor B)

Scope: owned root code only (`idna.go`, `idna_test.go`).

## Result: nothing clears the factorable bar

Only one real 2-occurrence pattern exists in the owned code, and it does not clear
the bar for a confident recommendation (3+ occurrences, or 2 substantial ones).

### Cluster A — err→(string, bool) collapse (found, NOT recommended)

**What it does**: converts a `(string, error)` result from the vendored profile into
this package's `(string, bool)` "ok" idiom.

**Occurrences (2)**: `idna.go` — `ToASCII` and `ToUnicode`, each:

```go
a, err := p.ToASCII(host) // or strict.ToUnicode(host)
if err != nil {
    return "", false
}
return a, true
```

**Proposed helper (considered, rejected)**:
`func okString(s string, err error) (string, bool)`, called as
`return okString(p.ToASCII(host))`.

**Impact**: ~10 lines → ~7 lines; net saving ≈1-3 LOC. Risk near-zero (mechanical
extraction, same behavior — see `audit-simplify.md` for the confirmation that the
error-path behavior this shape depends on is genuinely load-bearing, not incidental).
Not worth it: `ToASCIIErr` deliberately doesn't participate in this shape (it returns
the raw `(string, error)`), so the helper would only ever have two call sites, both
trivial, both in the same file. **Verdict: leave as is.**

## Considered and NOT recommended (surface-pattern-only, not real duplication)

1. **`t.Errorf`-based assertions across `idna_test.go`'s four test functions** — each
   is `if <condition> { t.Errorf(...) }`, but with different arities and
   purpose-built failure messages naming the exact failing case. A generic
   `assertEqual`/`check` helper would blur which case failed in test output —
   directly counter to attributable test failures. Standard idiomatic Go, not
   duplication.
2. **Profile selection** (`p := strict; if allowUnderscore { p = loose }`) — appears
   exactly once, inside `ToASCII` only (`ToASCIIErr` hardcodes `strict`, `ToUnicode`
   hardcodes `strict`). No second occurrence to factor against.
3. **Doc-comment phrasing** ("ok is false when...") repeated across `ToASCII`'s and
   `ToUnicode`'s comments — prose repetition, not code; out of scope for a dedup
   audit.

## Bottom line

Given the file set's size (~110 lines, 4 functions + 1 test file), "nothing to dedup"
is the honest, verified result — cross-validated independently by auditor A from the
simpler-pathways lens (see `audit-simplify.md`), which reached the same conclusion on
the same cluster.
