# Jet final review and release gates

Reviewed 2026-11-08. **The combined candidate is not ready for release.** Restricted rendering still exposes template-triggered panics, ordinary caught-error rendering fails, and the combined execution cost exceeds the existing 20% median-overhead target. Passing CI does not cover these new reproductions.

Scope: fork PRs [#1](https://github.com/ashtonian/jet/pull/1), [#2](https://github.com/ashtonian/jet/pull/2), [#3](https://github.com/ashtonian/jet/pull/3), [#4](https://github.com/ashtonian/jet/pull/4), [#5](https://github.com/ashtonian/jet/pull/5), [#6](https://github.com/ashtonian/jet/pull/6), [#7](https://github.com/ashtonian/jet/pull/7), and the two existing upstream PRs [#237](https://github.com/CloudyKit/jet/pull/237) and [#238](https://github.com/CloudyKit/jet/pull/238). Karsto adoption is excluded. This review changes no production source, PR description, branch history, module pin or remote publication. Internal plans and evidence remain fork-only under the [publication policy](plan.md#current-publication-policy).

## Reviewed inputs and evidence

| Input | Revision | Role |
| --- | --- | --- |
| Upstream v6.3.3 | `202464510d5b57f0afa9c5563f75babc1616682c` | Unchanged baseline |
| Correctness PR #237 | `822cf702300c6226fa56a3fcf2f8466dc6b9b712` | Separates initial writer/cleanup changes from later work |
| Combined candidate | `0e277b3017b4afae1f984663c4e9244b4de0b290` | Compiler, runtime, values/output, restricted profile, corpus and CI |
| Fork #7 review head | `2858cef45f7fdc9c100fe0294f367fea63c82a15` | Go files, dependencies, testData and public execution/compilation/restricted docs match the combined candidate |

Source snapshots were archived into isolated directories. Review probes and benchmark harnesses were added only to those snapshots. SHA-256 input manifests include untracked Go fixtures and corpus JSON. The new probes exercise public APIs with real templates and data, not mocked interpreter internals.

The combined candidate's existing [Linux CI](https://github.com/ashtonian/jet/actions/runs/42432265936) passes Go 1.16.15 normal, stable normal and stable race. This review additionally ran corpus conformance on all three snapshots, controlled legacy/restricted probes (the final defects reproduce in both normal and race builds), and normal-build benchmarks with separate CPU, allocation/live-heap, mutex and block profiles. See [evidence and reproduction instructions](final-review-evidence/README.md), [receipts](final-review-evidence/receipts.json), [PR descriptions at review time](final-review-evidence/pr-snapshot.json) and [benchmark data](final-review-evidence/results.json). Historical lane evidence is supporting material, not proof of the new cases.

## Findings

### F1 — P1: admitted data and builtin arguments can escape the restricted API as Go panics

**Reproduced defect; release blocker for restricted rendering.** Each case constructs a restricted Set with positive input, execution and compilation limits, gets its template, then calls `RenderRestrictedContext`:

| Template | Input | Actual result | Required behavior |
| --- | --- | --- | --- |
| `{{ .x }}` | `map[NamedString]string{"x":"ok"}` | String panic from `reflect.Value.MapIndex` | Return `"ok", nil`; the profile admits method-free string keys |
| `{{ .x = 1 }}` | `map[string]string{"x":"ok"}` | String panic from `reflect.Value.SetMapIndex` | Ordinary template type error; empty restricted result |
| `{{ .x = 1 }}` | `map[string]interface{}(nil)` | `runtime.plainError`: assignment to nil map | Ordinary template assignment error; empty restricted result |
| `{{ .[:] }}` | `[2]int{1,2}` | String panic: slice of unaddressable array | Preserve supported array slicing or explicitly define a compatible, checked operation |
| `{{ map("x") }}` | nil | String panic: incomplete key/value pair | Ordinary builtin argument error; empty restricted result |

The same five panic paths exist in the unchanged upstream baseline and the candidate's legacy API. They are **inherited evaluator defects exposed by the new restricted contract**, not five newly introduced regressions. The nil-map runtime panic can terminate an unprotected rendering goroutine/process; callers cannot rely on the returned error alone.

Owning operations:

- [Map lookup](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/eval.go#L1878) converts the key into `index` but calls `MapIndex(indexVal)` with the original unconverted value.
- [Assignment](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/eval.go#L349) performs map lookup/insertion without checking nil maps or key/value assignability.
- [Slicing](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/eval.go#L789) checks bounds but still reaches reflection with an unaddressable array.
- [`newMap`](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/default.go#L217) uses a string panic for bad arity.

Repair these owning operations with validated conversions and ordinary template errors. Preserve named types, caller snapshot isolation and catch behavior. Do not broadly swallow `runtime.Error` or non-error panics at the public boundary: trusted native programming faults have a separate, documented contract. The current identity-based builtin registry correctly controls authority, but an audited identity alone does not prove its argument semantics are safe.

Acceptance: all five public workflows pass their specified result/error assertions in normal and race builds; add nil/index/assignment and named-type variations around the repaired operations. Ordinary errors remain catchable, limits remain terminal, and deliberate native callback panics retain their behavior. No passing fix is claimed by this review.

### F2 — P2: the restricted profile rejects its own catch diagnostic

**Reproduced compatibility defect.** This valid existing workflow succeeds through legacy and unrestricted metered execution:

```jet
{{ try }}{{ missing }}{{ catch err }}{{ err }}{{ end }}
```

Restricted rendering returns `jet: capability denied: data with methods` and an empty string. [`executeTry`](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/eval.go#L669) binds the recovered Go error directly; [`beforeData`](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/restricted.go#L244) then rejects its `Error` method. A catch body that prints a literal works, so the corpus's rollback case misses this seam.

The owning layer should expose a bounded, data-only diagnostic for restricted catch variables. Keep arbitrary caller-supplied Error/Stringer/Formatter hooks denied. Specify the catch variable's supported representation and test printing, storing and using it in ordinary template data. Do not advertise unrestricted catch-variable compatibility until that is implemented or explicitly narrowed.

### F3 — P2: sink failures lose template location

**Reproduced diagnostic defect.** A writer returning a sentinel error from `receipt.jet` preserves `errors.Is(err, sentinel)` but returns only `"sink failed"` through both `Execute` and `ExecuteContext`. [`Runtime.recover`](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/eval.go#L278) gives the raw latched failure precedence over an annotated error; metered checkpoints can also raise the raw failure before node annotation.

Preserve the first terminal cause and attach bounded template/node context at its origin. Cleanup or a later error must not replace the cause. Check direct text, escaped/raw output, Renderer/SafeWriter paths and nested try commits for both inspectable cause and useful location. PR #237 already discloses this limitation; the combined candidate has not resolved it.

### F4 — P1 release gate: combined execution performance exceeds the agreed target

**Measured performance regression; structural optimization proposals remain unverified.** Six interleaved samples per snapshot, 15 fixtures in text and HTML, `GOMAXPROCS=2`, Go 1.27.3 darwin/arm64, normal builds, 100 ms per benchmark sample. Median below means the median of 30 per-fixture median-time ratios, not benchstat's geomean and not a production traffic-weighted aggregate.

| Comparison | Median time ratio |
| --- | ---: |
| Candidate legacy streaming / upstream legacy streaming | 1.67× |
| Candidate legacy streaming / correctness-only streaming | 1.25× |
| Candidate metered streaming, Background / upstream streaming | 3.12× |
| Candidate metered streaming, live cancellable context / upstream streaming | 3.16× |
| Candidate restricted owned string / upstream owned string | 3.22× |
| Candidate restricted owned string / candidate legacy owned string | 1.84× |

Owned-string baselines allocate a fresh buffer and return its string on each render. Streaming baselines reuse a buffer. Parsing and context creation are outside cached execution timing; input admission and restricted result ownership remain inside. All positive options are recorded in [candidate modes](final-review-evidence/candidate-modes.go.txt). Short samples contain outliers and sometimes wide intervals; [benchstat](final-review-evidence/benchstat.txt) preserves them. These are diagnostic results on one host, not final latency or production estimates.

Selected HTML cases, median ns/op:

| Fixture | Upstream stream | Candidate legacy stream | Candidate metered live | Upstream owned | Candidate restricted |
| --- | ---: | ---: | ---: | ---: | ---: |
| Header interpolation | 86.8 | 155 | 553 | 126 | 890 |
| Include | 234 | 658 | 922 | 277 | 1,284 |
| 8 KiB body | 6,293 | 6,523 | 6,930 | 8,036 | 9,099 |
| 64 KiB body | 50,514 | 51,288 | 51,146 | 62,723 | 66,235 |
| Escape expansion | 4,342 | 8,806 | 16,090 | 5,489 | 24,101 |
| Nested silent loops | 56,094 | 57,181 | 72,211 | 57,226 | 109,596 |

The previous four-fixture, runtime-only 1.08× result is valid only for its old candidate/configuration. It cannot serve as final acceptance of this combined implementation.

**Profiles identify two structural costs.** In the representative metered-live profile, `newParseAdmission`, `prepareSource` and `admitCached` account for 83.2% of sampled allocated bytes by their non-overlapping flat totals. Their corresponding share in the restricted profile is 50.7%, with `admitInputScopes` adding 19.1%. [Every metered render constructs admission state and walks the initial cached graph](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/execution.go#L197); [legacy cached includes also construct admission state](https://github.com/ashtonian/jet/blob/0e277b3017b4afae1f984663c4e9244b4de0b290/set.go#L164). Header interpolation grows from 24 B/1 allocation upstream to 1,000 B/11 allocations metered and 1,616 B/17 allocations restricted. Legacy include grows from 120 B/6 allocations to 968 B/13 allocations.

The other path is repeated small writes through formatting, escaping, checked output, reservations and the owned sink. Metered-live CPU shows `checkpoint` at 4.47% flat/12.78% cumulative and `checkedWriter.Write` at 4.47%/17.57%; `dataPrinter.value` is 20.77% cumulative. Legacy CPU also spends 7.14% flat in checked writes. These are overlapping stack percentages, not additive attribution of the total regression. Escape expansion is materially affected even in the unmetered API.

The first optimization packet should redesign cached-graph admission around immutable cost metadata plus small operation-owned visited/accounting state, preserving tighter callers, unique source/node charging, depth along every diamond path, cycles, development reparsing and namespace authorization. Investigate avoiding fresh maps for the common small graph and unlimited warm-cache path without retaining request contexts on Set/ASTs. The second should consolidate output ownership/checking so small escaped chunks do not repeatedly traverse redundant adapters. Preserve reservation-before-write, partial-write causes, first-error precedence, reentry safety and transaction accounting. Prove each change with one controlled source delta and matching fixtures before claiming an improvement. Do not fork the evaluator or remove checkpoints merely to hit the number.

[Profile reports](final-review-evidence/profiles.txt) include flat/cumulative CPU, allocation/live heap, mutex/block and process RSS. The serial profiles show little Jet contention; most sampled blocking is the benchmark harness's channel waits. They do not establish concurrent scalability. Runtime GC/syscall samples are substantial and are not evidence of a network bottleneck. Peak process RSS is roughly 14.8–15.8 MiB for these short profile processes; zero sampled live heap in several profiles is not zero retained memory. No heap/RSS cap is claimed.

Constructor-plus-initial-parse timing is retained [separately](final-review-evidence/compile-results.txt). Restricted setup includes source/global admission and registry construction; it is not directly interchangeable with a legacy Parse-only cost. Warm named lookup, cold dependency graphs, uncached parse+render, p50/p99, sustained concurrency and cancellation overshoot still require the final program below.

### F5 — P2: release packaging and review claims remain unfinished

**Verified unfinished work.** `go.mod` still declares `github.com/CloudyKit/jet/v6`. The proposed distributable fork identity and external resolution without `replace` have not been implemented or verified. The current PR heads also form a stack with prerequisite branches; merging #7 alone does not imply that all prerequisites reached the fork's main branch.

PR description accuracy at the recorded heads:

| PR | Assessment and required correction |
| --- | --- |
| Fork #1 — planning | Stale: says implementation has not started and describes an upstream-oriented submission sequence. Describe the original plan and link current candidate/review status; publication remains fork-only. |
| Fork #2 — values/output | Core accounting scope and native exclusions are supported. Narrow “invalid conversions return ordinary errors”: map lookup/assignment/slicing are not fully covered. Dependent policy code now exists, but F1/F2 and final performance still block acceptance. Move handoff chronology to evidence links. |
| Fork #3 — runtime | Synchronous ownership, sticky terminal state and no arbitrary callback preemption are supported. Its allocation/performance numbers are an old runtime-only diagnostic. Replace future upstream-submission language with the current hold; keep existing PR links historical. |
| Fork #4 — fixtures/CI | Correctly says baseline-only and draft. The latency/concurrency harness is not yet measuring the combined metered/restricted candidate. Replace stale waiting-on-implementation text with the concrete final gates. |
| Fork #5 — pull lexer | Focused lifecycle claim is supported by source and regressions. Preserves the third-party cycle prerequisite. Linked CI is integrated evidence, not an exact standalone-head CI claim; retain that distinction. |
| Fork #6 — compilation | Shared operation-local admission, synchronous loaders, cached-graph checks and exclusions are supported. Standalone head uses focused overlay evidence; combined full CI is disclosed accurately. Does not itself supply restricted execution. |
| Fork #7 — restricted profile | Denial/namespace/snapshot mechanisms are implemented, but F1/F2 prevent readiness. “Empty on failure” describes returned errors; it must not imply template input cannot escape as a panic. Invocation review is evidence of examined paths, not complete capability/semantic acceptance. |
| Upstream #237 — correctness | Current shortened description accurately discloses continued-work and lost-location limitations. The standalone patch remains incomplete at those boundaries; broader candidate checkpoints address the continuation problem, not F3. |
| Upstream #238 — runtime | Distinguishes runtime limits from sandboxing and discloses old benchmark scope. GitHub still targets upstream master, so its diff includes #237 rather than being mechanically based on that PR; body correctly states the prerequisite is included. Do not change it under the upstream hold. |

Recommended fork description format: concrete behavior in one paragraph, exact validation plus unresolved gates in a second, links for detailed evidence. Avoid a handoff log or a “production ready” claim. This review does not publish those edits.

## Architecture and preservation

The useful boundaries are already present: one evaluator, synchronous work ownership, one render-owned terminal state, separate compilation and input admission lifetimes, immutable private builtin identities, namespace authorization before cache use, input snapshots and an owned result. There is no need for a generic budget framework or another renderer. `try/catch` already existed upstream; this work changes its interaction with limits and output transactions.

| Contract | Implemented / verified status | Preserve during repair and optimization |
| --- | --- | --- |
| Legacy trusted API and Go 1.16 floor | Existing full CI passes; diagnostic regressions and inherited edge defects remain | Signatures, ordinary catch/rollback, native extension points, deliberate native panic behavior, text/HTML output |
| Shared execution/output/value limits | Existing boundary tests and combined corpus pass | Includes/exec/yield/content share state; rollback/discard receive no refund; first terminal cause survives catch/isset |
| Compiler/loader ownership | Implemented and existing normal/race CI verified | No detached worker; close before return; cache warmup cannot weaken later limits; ContextLoader remains cooperative |
| Restricted input/builtin authority | Implemented, existing denial cases pass; F1/F2 fail | Reject unused forbidden inputs, preserve named/typed-nil supported values, no caller mutation, aliases use sealed identity |
| Restricted result | Owned buffer and final copy charged; corpus passes | Empty returned result on error, no external writer callbacks, no residual borrowed references |
| Concurrent immutable templates/Sets | Existing CI race cases pass | No request state on shared Set/cache; callers still synchronize source/input mutation |
| Final performance | Current serial diagnostics fail target; full program incomplete | Match ownership/options; retain correctness; report per-fixture and tail costs |
| Fork distribution | Proposed, not implemented | Isolated module/import commit, external consumer proof, upstream ancestry/license/authorship |

Dependency review: [Jet v6.3.3](https://github.com/CloudyKit/jet/releases/tag/v6.3.3) remains the latest upstream release checked. Cycle [PR #236](https://github.com/CloudyKit/jet/pull/236) is consumed with its original authored commit. Trusted diagnostic [PR #234](https://github.com/CloudyKit/jet/pull/234) remains open and unconsumed; current `dump.go` still has its nil-context/unlocked-globals paths. That is an existing upstream-owned legacy issue, separately reported there; restricted `dump` is denied. Reuse its reviewed change if included in the fork's supported trusted workflow scope, rather than inventing a competing implementation or filing another issue.

`fastprinter` is pinned to `33d98a066a53`. Its [newer upstream change](https://github.com/CloudyKit/fastprinter/compare/33d98a066a53...1725d2651bd4cd68eeedef7e5a3e98a2954a9e99) accelerates Print/PrintPtr and raises its Go requirement from 1.13 to 1.20; Jet's examined paths use PrintValue/PrintString. A blind bump neither addresses the identified admission/output architecture nor preserves the current minimum Go contract. No dependency update is proposed without separate evidence and an explicit support-floor decision.

## Executable finalization sequence

These are consolidated work packets on existing fork branches, not new PRs or claims of scheduled/running agents. One implementer can execute them in order; independent documentation preparation can continue while a validation gate is queued.

| Packet | Existing review home / owner | Entry and work | Exit proof / blocked dependents |
| --- | --- | --- | --- |
| A — error semantics | Consolidated #7, with shared evaluator/builtin corrections carried into prerequisite branches as needed | F1–F3 reproductions; fix operations and restricted catch representation, preserving legacy native panic contract | Same public failing inputs pass normal/race; cause/location, snapshot, terminal and catch controls pass. Blocks release and final performance acceptance. |
| B — structural performance | Existing #6/#7 admission and #2/#3 output surfaces; measurements in #4 | Inspect retained profiles; implement admission-state and output-boundary improvements in separate attributable commits on the same task branch | Controlled old/new profiles and six-plus matched samples; preserve the acceptance matrix. Target <=1.20× median with per-workload/tail regressions visible. A missed target remains open, not an automatic waiver. |
| C — final benchmark/compatibility batch | Existing #4 against the exact consolidated A+B candidate | Freeze source, dependencies, harness, options and platform; run the program below once as a coherent batch | CI normal/race plus complete perf/overshoot evidence. Reuse unaffected completed stages; rerun invalidated ones after repair. Blocks readiness claim. |
| D — fork packaging | Separate commit in an existing consolidated fork PR | After generic code acceptance: rename module/self-imports/examples/submodules to proposed fork coordinates; preserve licenses/authorship | Fresh external consumer resolves the exact pushed revision without `replace`, with no accidental second Jet module; minimum/current Go CI passes on packaged SHA. No release tag is invented. |
| E — final review handoff | Existing descriptions and fork plan | Accurate current scope, exact heads/bases and remaining limitations; ensure no internal docs enter upstream diffs | Reviewed merge/dependency order and signed/verified commits. Any eventual merge/tag/release is a distinct publication action, not performed by this review. |

A is localized but affects core reflection semantics; B spans admission and output ownership and needs controlled measurements; C/D cover multiple toolchains and module boundaries. Scope is identifiable, but an effort estimate would be low confidence. No additional PR is required. Do not close/rebase/rewrite existing PR history merely to consolidate the presentation.

### Final benchmark program

1. Preserve the upstream and correctness-only baselines. Compare final legacy streaming, metered streaming with both Background and live cancellable contexts, and restricted owned strings against matched owned-string baselines. Keep real positive budget values fixed and assert exact text/HTML output before timing. Include valid cases near their bounds plus separate bounded rejection workloads.
2. Measure constructor/source admission, cold compile and dependency loading, warm named lookup, cached render, and uncached parse+render separately. Cover the existing 15 fixtures plus data-heavy maps/arrays/JSON, accepted named/typed-nil values, builtin aliases and corrected catch-variable workflows. No Karsto endpoint or helper acceptance is implied.
3. Use at least six interleaved independent normal samples, increase duration from this diagnostic 100 ms to at least 1 s for release comparisons, record host/toolchain/go env, and use benchstat intervals. Profile representative paths separately. Report ns/op, B/op, allocations, throughput, CPU and peak RSS; compare alloc_space with live heap rather than equating either with RSS.
4. Adapt the existing #4 latency/concurrency harness to all candidate modes. Clock-sample p50/p99 separately from throughput; run fixed workers 1/2/8 and a host-fit maximum, shared templates and separate Sets, matching GOMAXPROCS. Collect mutex/block profiles under that load. Detect harness saturation before inferring equal capacity. Avoid shared mutable vars/results in worker loops.
5. Measure cancellation overshoot separately for native interpreter loops, prepaid builtin scans, compile/input admission and cooperative trusted callbacks. Use bounded finite inputs and synchronized cancellation; record cancellation-request time, return time and any post-return activity. Native callbacks that are intentionally blocked must be explicitly released and joined. Restricted mode must never invoke them. Pair wall-clock distributions with deterministic semantic controls; no flaky timing assertions or new fuzzing.
6. Prefer exact-SHA Linux CI for full minimum/current-Go normal and current-Go race suites. Keep race figures out of performance tables. Run required Darwin-only linker/platform checks locally. Every admitted stage has a deadline, bounded TERM/KILL cleanup, source/config manifest, durable log and result receipt. Benchmarks reserve both host lanes with `gate-lock run-exclusive`; check-only analysis uses `run-check` and named-package regressions use `run-focused`.

The final handoff needs passing A, accepted B/C evidence and D's external-resolution proof. No unresolved contract requires a speculative framework: preserve the currently stated supported surface and repair its owning layers. Changing the support floor, dropping accepted values/catch behavior, relaxing the performance target or publishing a release would be explicit decisions rather than hidden changes.
