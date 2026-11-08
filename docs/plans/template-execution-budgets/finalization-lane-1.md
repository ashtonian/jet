# Finalization lane 1: evaluator and catch repairs

Base: `0e277b3017b4afae1f984663c4e9244b4de0b290`. Task branch: `work/finalization-lane-1`. Scope is F1/F2 and the inherited trusted dump defects under the 2026-11-08 finalization packet. This evidence stays fork-only; it does not authorize publication, merging or tenant adoption.

## Repairs and compatibility

- Map reads use the checked, converted key. Named method-free string keys retain their admitted types. Dynamic interface keys are checked for hashability without invoking user methods.
- Assignment to the parsed `_` discard target is a no-op, including assignment-form ranges. This prevents the accepted discard target from reaching a FieldNode assertion.
- Field/map writes validate non-nil targets, accessibility and assignable values before reflection. Invalid untyped-nil field/map writes are ordinary errors, not map deletion syntax. Existing valid assignments remain supported; restricted writes affect only the admitted snapshot.
- Slicing validates the base and numeric bounds. Unaddressable arrays gain private addressable backing storage after the full payload/work reservation. Addressable arrays/slices retain aliasing. Fractional numeric indices retain truncation toward zero; overflow/NaN/invalid combinations return ordinary errors.
- `map`, `slice` and `array` reject invalid/inaccessible arguments as ordinary errors. Map keys and values are evaluated once; convertible legacy keys use their validated conversion. Assignable typed nil collections remain valid values.
- Restricted catch bindings are copied plain strings capped at 512 bytes, with work/value reservation before construction and UTF-8-safe truncation. Only private interpreter-owned message text or a directly recovered string is read. Unknown errors use `template execution failed`; their Error/Stringer methods are never called. Trusted catch bindings remain Go errors. `%w` cause chains, terminal failures and deliberately escaping native programming panics are preserved.
- Trusted dump adopts CloudyKit/jet #234 as signed commit `4deac706140619086f08bd87dbc8f9d4aedc077e`, preserving youdie006's authored patch from `f300a9af0cdcf75d9ea90516d8f915b45bd127f9`. It guards nil context and snapshots globals under RLock before formatting. A small follow-up also handles nil global values in both dump forms. Restricted dump remains denied.

The private `templateError` marker requires minimal cross-boundary edits in `NodeBase.error/errorf` and `Arguments.Panicf`; its Error text/Unwrap cause preserve ordinary trusted behavior. Lane 2 must preserve this marker through any sink-location changes. No Runtime recovery, sink, terminal or reservation implementation is changed here. The sibling thread-message transport was unavailable, so the exact seam is recorded in `.git/finalization-handoff.json` for integration.

## Controlled evidence

Durable artifacts are retained at `.git/finalization-lane-1/` in this checkout. Each stage has a JSON receipt with command, source/dependency/config hashes including untracked fixtures, toolchain, status and timing, plus a log. Sources were stable during admitted checks. Queue wait is separate from execution; installed gate-lock applies 120–240-second external deadlines and five-second TERM cleanup. Go test's own timeout is an additional harness guard.

| Stage | Observation |
| --- | --- |
| `baseline-normal` | Public legacy/metered/restricted probes reproduce named-key, bad/nil assignment, array-slice and malformed-map failures. Restricted caught errors are denied as method-bearing data. Trusted nil-context dump fails. |
| `baseline-race` | Independently reproduces the inherited unlocked-globals race with `GORACE=halt_on_error=1`; this deliberate failure exits before the semantic cases. |
| `baseline-semantics-race` | Separate race run reproduces F1/F2 with the same inputs, without the dump race stopping the batch. |
| `fixes-first` | The original public reflection/catch cases and authored dump tests pass. |
| `range-discard-baseline` | Accepted assignment-form ranges with `_` escape as runtime.TypeAssertionError in all three APIs; declaration-form controls succeed. The minimal executeSet guard is tested in the complete stages. |
| `adjacent-first` | Nil-global dump remains broken after #234; this is the baseline for the small follow-up. One fractional-slice fixture incorrectly used Jet's reserved `end` keyword and was corrected to `stop`; it was not an evaluator failure. |
| `complete-normal`, `complete-race` | Expanded named-package checks pass on current Go: reflection/catch/builtin/dump cases plus existing runtime, cancellation/cause, programming-panic, terminal, policy-denial, parser and ordinary rollback controls. |
| `complete-go116`, `typed-nil-controls-race` | Minimum-Go focused and typed-nil positive controls pass. |
| `compatibility-root-normal`, `compatibility-root-race`, `compatibility-root-go116` | Complete root-package suite passes on Go 1.27.3 normal, separate race and Go 1.16.15 external-link normal, including the original evaluator fixtures. |
| `checks-complete` | Named-package go vet and documentation/formatting checks pass. |

Final stage commands and results are authoritative in their receipts. Go 1.16.15 uses `-ldflags=-linkmode=external` on this modern Darwin host; retained linker warnings are distinct from source/test failures. No dependency or support-floor change is introduced. The finalization handoff lists completed/pending checks explicitly; combined minimum/current normal/race CI remains lane 4's integration gate.

Tests exercise public Template APIs with real admitted inputs and constructor arguments. Additional pure controls demonstrate no native hook invocation during catch conversion, correct exact-boundary array reservations and UTF-8 truncation. Snapshot, Error wrapping and deliberate native-fault controls prevent repairing template mistakes by broadly swallowing Go panics. No dedicated fuzzing or dangerous allocation probe is added.

## Integration and remaining acceptance

Consume the authored dump commit followed by the signed repair commit recorded in `.git/finalization-handoff.json`. Changed evaluator regions are executeSet, executeTry catch binding, the slice expression, resolveIndex and indexArg; new helpers own reflection/diagnostic semantics. default.go changes are confined to newMap/newSlice. Public docs are restricted.md and execution-costs.md. Preserve these regions when integrating lane 2's output/terminal work and lane 3's admission changes.

Lane 1 does not claim a performance improvement, complete combined acceptance or packaged module resolution. The retained final-review profiles/options were read; structural output/admission optimization and the final benchmark/packaging gates belong to lanes 2–4. No upstream write, new PR, force push, release or Karsto change is performed.
