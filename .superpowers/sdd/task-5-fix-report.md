# Task 5 Fix Report

## Change
Switched the Solitaire engine RNG contract from `math/rand/v2` to the standard `math/rand` package required by the Task 5 brief and downstream UI contract.

## Changed files
- `internal/solitaire/engine/deal_test.go`
  - Updated deterministic test RNG to `rand.New(rand.NewSource(1))` from `math/rand`.
- `internal/solitaire/engine/deck.go`
  - Updated shuffle helper import and `*rand.Rand` type to standard `math/rand`.
- `internal/solitaire/engine/game.go`
  - Updated `New(rng *rand.Rand)` import/type binding to standard `math/rand`.
- `.superpowers/sdd/task-5-fix-report.md`
  - Added this fix report.

## Tests
- `go test ./internal/solitaire/engine` — passed.

## Commit hash
- Code fix: `238cfd5338d2ca7f9381bda4bf298eb7c9da3780`

## Self-review
- Confirmed the red state first by changing the test helper to standard `math/rand`; the package failed to build because `New` still required `*math/rand/v2.Rand`.
- Kept the change limited to RNG imports/types and the deterministic seed helper.
- Re-ran the targeted Solitaire engine test package after production changes.
- Checked `internal/solitaire/engine` for remaining `math/rand/v2` and `NewPCG` references; none remain.

## Concerns
- No behavioral changes were made beyond the RNG implementation stream changing from `rand.NewPCG(1, 2)` to `rand.New(rand.NewSource(1))` for deterministic tests, as required by the contract.
- Pre-existing untracked docs files remain outside this fix and were not modified or committed.
