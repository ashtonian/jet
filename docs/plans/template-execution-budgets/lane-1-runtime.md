# Lane 1: runtime ownership and metering

**Execution started 2026-11-07** under the owner's explicit lane-start instruction, superseding the prepared hold. Use one local implementer for diagnosis, code, self-review and verification; do not spawn reviewers or subagents.

## Read first

[Plan and shared-file ownership](plan.md), [source audit](source-audit.md), [RFC-0001](../../rfcs/0001-execution-lifecycle.md), [RFC-0005](../../rfcs/0005-fork-and-upstream-strategy.md), [runtime validation](validation.md), and current repository/global coding guides. Refresh upstream master and relevant PRs; baseline v6.3.3 is `202464510d5b57f0afa9c5563f75babc1616682c`.

## Assignment

Deliver J01–J03 as compilable, upstream-reviewable PRs. Own the execution budget and terminal failure mechanism; no other lane builds an alternate one. Preserve `Execute`, syntax and unrestricted extension points. Public `ExecuteContext` remains synchronous; no timeout worker/select.

1. Reproduce self-yield failure in a bounded subprocess, silent computation, ignored sink errors and retained runtime references. Keep independent causes distinct.
2. Repair cause wrapping, relevant writer failures and pooled-state cleanup as independent correctness patches where separable. Inspect every recover and resource acquisition, including rangers and try scope restoration.
3. Implement context checkpoints and sticky terminal unwind through both try and isset. Handle entry, completion, empty templates and call-return checkpoints. Specify panic compatibility without hiding programming faults as budget errors.
4. Implement fuel/depth across statements, loop advances, recursive expressions/value comparison, blocks/content/yield/include/exec and inheritance traversal. A single budget belongs to one render. Bounded values and capabilities are later protections, not claims this PR makes.
5. Publish the internal metering/reservation/terminal API for lanes 2/3. Do not expose unenforced value/output fields yet. Context should flow explicitly; any execution-local adaptation to existing callback signatures must be documented and cleared.
6. Run your own R01–R12 and upstream compatibility/race coverage, review performance profiles for introduced dispatch costs, and update the RFC/PR with actual exclusions.

## Handoff contract

Lane 2 needs checked remaining-budget reservation, fatal error propagation, a try/output transaction hook, and nested-call state. Lane 3 needs policy checks before native invocation, a compilation-context bridge and a terminal denial path. Agree those contracts in code/godoc and test them; an invented placeholder interface is not completion.

Shared evaluator changes from other lanes wait for the recorded foundation SHA, then arrive as small dependent PRs. Send only concrete contract/patch references; do not delegate your self-review. Preserve upstream fixes and third-party authorship.

## Done means

- [x] Controlled baseline and fixed interpreter cases pass under matching inputs, with externally bounded dangerous tests.
- [x] No owned interpreter work or pooled caller references after return on success/error/cancel/panic.
- [x] All native loop/recursion entry points have a metering disposition; try/isset cannot swallow termination.
- [x] Legacy compatibility and positive-limit API tests pass, causes remain inspectable, race results recorded separately.
- [x] J01–J03 descriptions state that values, compilation and arbitrary native callbacks remain outside this milestone.

Do not merge a tenant-safe claim or close Karsto #209 from this lane's results. Use the shared host gate/deadline/log rules in the plan; prefer CI for full validation. Preserve scope while a stage runs and resume only invalidated/failed checks.

## Published foundation

Dependent work can use **`d2f089c`** on `work/lane-1-runtime` ([fork PR #3](https://github.com/ashtonian/jet/pull/3)). Earlier `86fce7a` introduced the API; `388f288` fixes public YieldBlock restoration; `d2f089c` adds numeric-scan preflight and reusable sink adapters. The internal method signatures below remain stable across those commits. Do not assume remote master includes them.

- `checkpoint(uint64)`, `enter()/leave()`: cancellation, fuel and recursive-frame ownership.
- `reserve(remaining *uint64, limit, requested uint64, kind string, cause error)`: checked pre-work subtraction, zero unlimited, sticky terminal failure with an inspectable LimitError. Lane 2 extends executionState/options only when its dimensions are enforced.
- `terminate(error)`, `rethrowTerminal()`, `rethrowRecovered(interface{})`: first-error latch through try/isset and callback recovery. Programming panics are not reclassified as limits.
- `beforeCall(reflect.Value)`: actual pre-invocation checkpoint for Func/reflection, Renderer/printing, SafeWriter and Ranger calls, paired with a return checkpoint. Lane 3 extends capability admission here and covers implicit formatting/configured escape callbacks in its capability inventory; no restricted policy is implemented by this foundation.
- `executionContext()` and `loadTemplate(name,parent string,sibling bool)`: actual runtime compilation bridge. Lane 3 replaces the legacy load inside this seam with shared compilation admission, without moving request state onto Set/Template/cache.
- `beginOutputTransaction()/commitOutputTransaction(*outputTransaction)`: actual try buffering/commit hook in execution_output.go. Lane 2 adds pre-growth value/output reservations, rollback accounting and commit transfer without double output charge here.
- `exec` and `includeIfExists` in default.go already use loadTemplate/templateRoot and the current Runtime. Lane 2 can now own their dependent builtin changes against the explicit SHA.

The context stored in executionState is an execution-local adapter to existing Runtime/Arguments callbacks, cleared with every pool return. Nested interpreter calls reuse it. Public Execute/ExecuteContext reentry from arbitrary host code starts a new top-level execution and must be denied by restricted policy.

Evidence and remaining limits: [lane-1-evidence.md](lane-1-evidence.md). The generic correctness submission is [CloudyKit/jet #237](https://github.com/CloudyKit/jet/pull/237), mapping canonical `3863ed9`/`388f288` to submission `8b5e7b0`/`822cf70`; its diff contains no internal documents or packaging. J02/J03 remain one coherent context/fuel/depth API proposal.

Lane 3 peer compile commit `9a8d22e` changes the private sibling-loader signature to accept parseAdmission. Its dependent runtime patch must replace loadTemplate's load with one render-shared parseAdmission using executionContext(), not create a fresh compilation allowance for each include/exec. The peer code is reviewed, not integrated into this lane; compilation remains explicitly excluded until that dependency lands.
