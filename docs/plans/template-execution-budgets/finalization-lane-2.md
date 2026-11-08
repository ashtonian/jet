# Output finalization: F3 and structural cost

Current follow-up code: `ec0221fb7ecd4f5416a289b8e019d60c65793872`, based on combined `54b561484e1465edea963ed04579bcd24a7dbf76`. Protection and diagnostic performance validation pass; the complete integrated release target, CI and packaging remain open. The earlier `5ba5bda` handoff is historical. This report does not grant final acceptance.

Historical F3 base `0e277b3017b4afae1f984663c4e9244b4de0b290`; code `f599fc95992f104dd6e8cab180df04c9f964b5fa` on `work/finalization-lane-2`. The original output commit `56d25c8` and ready handoff `f7c2b675e1c88f7731276efeece790ad5e5fbaea` contained a scalar scratch borrowing regression and are superseded by the repair below. Focused correctness and affected measurements now pass; the repaired lane is ready for integration. **Combined performance, packaged CI and external-resolution acceptance remain open.** Nothing was published upstream, no new PR was created, and existing review branches were not rewritten.

## Behavior and preservation

The first sink failure now carries bounded template/node context and preserves `errors.Is`/`errors.As`. Nodes with line metadata report it; upstream text nodes without it report the exact starting byte offset. Context is created on failure only. A later limit/error/cleanup cannot replace the first observed terminal cause. Native programming-panic behavior remains covered by the existing controls.

One checked destination performs post-escape generation accounting, retained-value reservation and error latching. It replaces the escaping check adapter, executionWriter adapter and restrictedSink reservation callback. Root/transaction sinks and the root scope/printer are reused within Runtime and cleared before pooling. Scalar printing leases bounded scratch through the complete Write call; overlapping print invocations own separate workspaces. Known `*bytes.Buffer` destinations append directly; arbitrary writers keep partial/error/short-write handling. Trusted redirected destinations receive checked escaping/commit fallbacks.

Try commits copy the destination's transfer mode, preserving generation accounting on the original writer during native reentry. Newly retained copies still consume value allowance; rollback/discard refund neither allowance. Includes/exec share the same state. Background/TODO avoid impossible cancellation checks; custom/live contexts retain their Err checkpoints. No budget setting, ownership contract, supported fixture or capability restriction was relaxed. No extra output-buffer pool, initial reserve or caller tuning option was selected.

## Historical output correctness (`56d25c8`)

The base public sink workflows failed with the original cause but absent location; the fixed workflows pass. Coverage includes text, escaping, raw/safe/JSON, Renderer/SafeWriter ignored errors, partial writes, short writes, nested includes/try commits, reentrant escapers/commit generation, redirected writers and owned results across a large-then-small/post-GC sequence.

The initial first-cause probe wrote into an uncommitted try buffer; its step failure was not evidence of cause replacement and is excluded. Its corrected public workflow writes to the real sink. An early reentry probe needed to account for automatic block-body execution; an early short-write probe incorrectly treated an empty successful write as failure. Those harness failures are retained locally. Redirected-writer coverage caught an intermediate consolidation regression (success despite ignored sink error); the matching fixed probe passes.

| Historical output-source check (`56d25c8`) | Evidence | Result |
| --- | --- | --- |
| Current normal, runtime/value/builtin/restricted/corpus and new output cases | [direct-buffer-controls](finalization-lane-2-evidence/direct-buffer-controls/result.json) | Pass |
| Current race, same selected scope | [selected-output-race](finalization-lane-2-evidence/selected-output-race/result.json) | Pass |
| Darwin Go 1.16.15, same selected scope | [selected-output-go116](finalization-lane-2-evidence/selected-output-go116/result.json) | Pass; old LC_DYSYMTAB linker warning retained |
| Vet | [selected-output-vet](finalization-lane-2-evidence/selected-output-vet/result.json) | Pass |
| Original F3 baseline | [sink-baseline](finalization-lane-2-evidence/sink-baseline/run.log) | Expected failure; cause preserved, location missing |
| Redirected-writer control/fix | [baseline](finalization-lane-2-evidence/redirected-writer-baseline/run.log), [fix](finalization-lane-2-evidence/redirected-transaction-fix/run.log) | Expected failure then pass |

Installed host-gate admission bounded every stage. Focused checks use `-p=1 -parallel=2`, `GOMAXPROCS<=2`, explicit 60/120-second test timeouts and 120/180-second execution bounds; TERM escalation is five seconds. Exclusive measurements have child deadlines/cleanup and a separate outer execution bound. Queue waiting is separate. No dedicated fuzzing, OOM or runaway probe was used; the shared amplification regressions remain finite (largest result 128 KiB).

## Historical selected output comparison (`56d25c8`)

Six interleaved one-second normal samples alternate baseline/selected order on Go 1.27.3 darwin/arm64, GOMAXPROCS=2. Header interpolation and escape expansion run in both text and HTML modes. Positive execution and compilation options, fixtures and result ownership are unchanged. Background, live-context and restricted modes are separate; restricted returns an owned string in both builds. The baseline is the **combined candidate**, not upstream. [All samples/medians](finalization-lane-2-evidence/selected-measure/summary.json), [benchstat intervals](finalization-lane-2-evidence/selected-measure/benchstat.txt), [commands/binary hashes](finalization-lane-2-evidence/selected-measure/results.json).

HTML escape expansion:

| Mode | Combined base → selected median | Time change | Allocations/op |
| --- | --- | --- | --- |
| Legacy stream | 8.772 → 6.389 µs | -27.17% | 1 → 0 |
| Metered Background stream | 15.743 → 8.878 µs | -43.61% | 11 → 7 |
| Metered live stream | 16.133 → 10.515 µs | -34.82% | 11 → 7 |
| Restricted owned string | 24.374 → 13.041 µs | -46.50% | 23 → 19 |

Each escape time change has benchstat p=0.002, n=6. Other sampled timing changes were not statistically detected; do not claim header speedups from their surface medians. Legacy sampled output has zero hot allocations; metered header allocation is 1,000→848 B/op, 11→7 allocations; restricted header is 1,616→1,448 B/op, 17→13. These are measurements, not stable allocation assertions or quota units.

Matched CPU profiles use default memory sampling, separately from normal timing. Flat/cumulative/source views show one checked destination and removal of the extra dispatch/terminal layers. Allocations at Runtime.recover/root scope, outputWriter/root setup and dataPrinter are removed from that sampled non-reentrant hot render path. The remaining metered allocation profile assigns 95.81% of sampled bytes to compilation admission (unchanged in this lane). Residual live-context escape CPU includes checkedWriter.Write (19.82% flat) and reserveWork (11.06% flat, including context checks); overlapping percentages are not additive. These are costs for the combined integration to reassess, not waived gates.

Raw CPU, allocation/live heap, mutex and block reports are linked under [selected-measure](finalization-lane-2-evidence/selected-measure/profile-inspection.json). Contention is chiefly runtime/harness activity in this serial profile, not a concurrent-scaling proof. Peak RSS (base→selected): legacy 14,532,608→14,270,464 B; live metered 14,925,824→15,187,968 B; restricted 15,253,504→15,384,576 B. Zero sampled live heap in a report does not mean zero retained memory. No heap/RSS cap follows from logical quotas.

## Object-count completion

All existing heaps were reused to add **88 flat/cumulative alloc_objects and inuse_objects views across 22 captures**: six selected/control, five strategy plus five large-then-small, and six historical first-consolidation captures. [Inspection commands](finalization-lane-2-evidence/object-profile-inspection.json), [unchanged capture/binary hashes](finalization-lane-2-evidence/object-profile-source-proof.json) and [view summary](finalization-lane-2-evidence/object-profile-summary.json) preserve provenance. That supplement replayed no benchmark or capture and left production source unchanged; the subsequent scratch repair supersedes its source/readiness SHA. Object views use each original binary/profile pair; the prior CPU/source views retain their frozen-source mapping.

The allocated-object views confirm removal of frequent small output allocations, independently of byte totals. Base legacy attributes 91.70% of sampled allocated objects to Runtime.recover's fresh scope; that site disappears from the selected hot path. Base live metered shows outputWriter, dataPrinter.value/printValue and recover allocations; those sites disappear after reuse/consolidation. In the selected live capture, the disjoint admission sites account for **99.10% of allocated objects versus 95.81% of allocated bytes**. These profiles contain the pre-integration compiler/input implementation; lane 4 must use fresh integrated profiles to assess lane 3's improvements.

Restricted selected capture, disjoint flat sites or their equivalent cumulative owner:

| Allocation owner | Allocated objects | Allocated bytes | Implication |
| --- | ---: | ---: | --- |
| Compilation admission | 35.53% | 6.49% | Frequent small accounting objects remain in this source. |
| Input admission | 18.12% | 3.87% | Byte attribution alone understates object/GC pressure. |
| Buffer growth | 36.36% | 65.27% | Growth dominates bytes more than object frequency. |
| Owned string copy | 4.97% | 23.57% | A small number of larger copies preserves result ownership. |

Percentages are within-profile attribution, not additive CPU costs or per-render counts. Profiling calibrates different iteration totals in each process, includes setup/harness allocations, and default memory sampling statistically weights objects. Do not compare absolute process object totals as if the render counts matched. The normal benchmark's B/op/allocs/op and confidence intervals remain the per-operation evidence.

Sparse live-object samples need particular care: five of six matched default-sampling inuse_objects views report zero; the selected legacy view contains an estimated 1,092 objects at a runtime.mallocgc→malg site. This is a sample-weighted runtime estimate, not a census of goroutines or evidence of a Jet retention regression. The rate-1 large-then-small screen provides a more specific observation: the reuse variant retains **one 4 KiB scratch object** at checkedWriter.Write, corroborating its live-byte view, while natural growth has no such site. Ten of 44 generated live views have zero sampled totals. None proves zero live memory, absence of leaks, or an RSS bound.

The object screen also confirms that reuse removes sampled growth objects but leaves input/admission objects and the owned string copy. This does not change the strategy verdict: normal timing showed no supported warm serial CPU win, and the cold/shared-context screen caveats below remain in force. Screen CPU uses memprofilerate=1 and includes profiling pcvalue/step overhead; the selected CPU profile/default-sampling and normal timings remain separate. A summary-parser relative-path error was repaired and only the summary step resumed; all generated object views were reused.

## Historical buffer strategy screen (`56d25c8`)

A separate short screen changes one factor at a time: natural growth, capped 256/1024/4096-byte first-write reserves, and cleared 4096-byte Runtime reuse. Each variant has frozen source/binary hashes and identical positive limits. The hints occur after actual logical reservation, skip empty/large first chunks and never preallocate a quota. Reuse never retains a grown output array and copies returned strings before recycling scratch. No rejected variant is production code.

| Strategy | Tiny B/op | Escaping B/op; allocations | Successful try B/op | Verdict |
| --- | ---: | ---: | ---: | --- |
| Natural | 1,072 | 12,564; 19 | 1,592 | Selected |
| 256 B hint | 1,264 | 12,374; 17 | 1,976 | Reject: small/rollback cost, no supported CPU gain |
| 1 KiB hint | 2,032 | 11,603; 15 | 3,513 | Reject: small/rollback cost |
| 4 KiB hint | 5,104 | 8,530; 13 | 9,659 | Reject: substantial small/rollback cost |
| Cleared 4 KiB reuse | 1,008 | 4,434; 12 | 1,529 | Reject: no warm serial CPU gain; retained-memory/cold tradeoff |

[Screen values](finalization-lane-2-evidence/reserve-screen-summary.json), [reuse values](finalization-lane-2-evidence/reserve-reuse-summary.json) preserve three 200 ms samples per row. Reuse's warm escape/tiny/header medians were 1.023×/1.025×/1.045× natural; successful try was 1.137×. These short sequential samples select no CPU win. The large-then-small live profile observes 4 KiB retained by the reuse variant; it does not retain the preceding 64 KiB output. A per-entry cap is not a global memory bound.

Empty/tiny, escaped, 8/64 KiB, success/rollback, 2/8-worker, post-GC and large-then-small controls were executed. **postGC times runtime.GC()+render, not isolated cold-render latency. Parallel screens share one cancelCtx across workers and include context synchronization/lock costs.** They do not isolate scalability; lane 4's final harness uses worker-owned live contexts. Screen CPU profiles use memprofilerate=1 and include pcvalue/step allocation-profiling cost; normal timings are separate. CPU, allocations/live heap, large-small live heap, mutex/block and RSS were inspected for every variant. No rejected variants need replay for acceptance.

## Evidence and integration

Portable text is an adapted copy: host paths are replaced, other owners' queue notices omitted, trailing whitespace normalized. [Raw hashes](finalization-lane-2-evidence/raw-artifact-sha256.json) identify original local evidence under `.git/finalization/`; binaries and pprof stay there. [Toolchain/dependencies](finalization-lane-2-evidence/toolchain.json), selected build/check receipts, exact variant sources and replay drivers are committed. Initial host-core-inherited samples were stopped/excluded. Early profile source views resolved the current checkout; corrected views use each frozen source snapshot and replace those views.

Integrate lane 1, then lane 3, then this lane. Preserve lane 1's NodeBase/Arguments templateError marker and catch/assignment changes. Preserve lane 3's executionState.compilationState field, full reset, and the restricted input block's borrow/defer-release/admitInputScopesInto hunk before Runtime acquisition. This patch changes executionState terminal/checkpoint functions and output setup, without editing startCompilation/loadTemplate, catch, map/slice, input or policy selection.

The code commit is locally GPG verified with the effective configured identity. GitHub verification follows lane 4's authorized fork publication. Exact combined normal/race/minimum-Go CI, representative upstream-relative <=20% median target, p50/p99/concurrency/overshoot and packaged external module resolution remain lane 4 gates. The subset above does not meet or waive them; small-template and residual escaped-write costs remain visible. No release/readiness/sandbox claim is made.

## Scalar borrow lifetime repair

Code `f599fc95992f104dd6e8cab180df04c9f964b5fa` fixes lane 4's public probe `fae8b0a1a3f16a12820c63d9517b692c2e8def24`: a writer borrowing formatted Go int 123 reenters Runtime.YieldBlock, whose int 456 formatting overwrote the same backing array in `56d25c8`. Restoring the printer after nested return was too late: the nested writer already observed corrupted borrowed bytes. Numeric template literals take the float path and did not expose it. The earlier ready SHA remains explicitly invalidated.

The exact signed probe was consumed with its original author via signed cherry-pick `0fb28eea333e524518428149ca25013440eaf3ff`; its SHA-256 is `cd11dbdf2249e6c36b35171391c023f284be156d80444d3fafad0519bbb00e42`. It fails identically on normal, race and Go 1.16.15 with the old implementation (borrowed chunk becomes 456), then passes with the fix. Additional public tests hold three levels of borrowed chunks for int, uint, bool, named int, collections, float and string in Background/live contexts.

The root printer is reusable only while idle. Each overlapping print invocation receives a distinct workspace until its own Write returns, then clears it. Sequential calls retain root reuse; reentrant calls own their backing array. The repair changes neither value/output/work quotas nor callback lifetime. The earlier scratch restoration assertion is superseded by this lease invariant.

| Repair check | Evidence | Result |
| --- | --- | --- |
| Exact baseline, normal/race/minimum Go | [normal](finalization-lane-2-evidence/scalar-borrow-baseline/result.json), [race](finalization-lane-2-evidence/scalar-borrow-baseline-race/result.json), [Go 1.16](finalization-lane-2-evidence/scalar-borrow-baseline-go116/result.json) | Expected failure in both contexts |
| Exact and adjacent public borrowed-workspace regressions | [focused](finalization-lane-2-evidence/scalar-borrow-fix/result.json) | Pass |
| Current normal/race, runtime/output/value/builtin/restricted cases | [normal](finalization-lane-2-evidence/scalar-borrow-fix-normal/result.json), [race](finalization-lane-2-evidence/scalar-borrow-fix-race/result.json) | Pass |
| Same scope, Go 1.16.15 Darwin arm64 | [minimum Go](finalization-lane-2-evidence/scalar-borrow-fix-go116/result.json) | Pass; existing external-linker warning retained |
| Vet | [vet](finalization-lane-2-evidence/scalar-borrow-vet/result.json) | Pass |
| Committed-tree probes and output cases, temporary benchmark removed | [receipt](finalization-lane-2-evidence/scalar-borrow-committed-tree/result.json) | Pass |

No full suite was replayed. The normal/vet receipts retain `inputs_unchanged=false` because the report changed during execution; [source proof](finalization-lane-2-evidence/scalar-repair/source-proof.json) separately verifies every Go/module/fixture hash against the repaired tree and preserves the non-source changes. The race/minimum-Go and committed-tree source inputs also match. The ignored benchmark is archived, excluded from production and absent from the final tree. Installed gate admission, execution/cleanup bounds, commands, toolchain, input manifests and logs are retained for every stage.

Affected scalar measurements use identical positive quotas (1,000,000 steps, depth 64, 8 MiB value, 1 MiB output), typed Go inputs, exact output assertions and GOMAXPROCS=2. Six interleaved one-second samples compare the old source only on sequential workflows that produce valid output. Reentry uses a safe invocation-local workspace control on otherwise identical source; corrupt output is never treated as an equivalent result. [Build/source proof](finalization-lane-2-evidence/scalar-repair/source-proof.json), [raw samples](finalization-lane-2-evidence/scalar-repair/summary.json), [sequential intervals](finalization-lane-2-evidence/scalar-repair/sequential-benchstat.txt) and [reentry intervals](finalization-lane-2-evidence/scalar-repair/reentry-benchstat.txt) identify the comparisons.

| Workflow | Control → repaired median | B/op; allocs/op | Timing inference, n=6 |
| --- | ---: | --- | --- |
| Sequential int, Background/live | 480.9 → 471.3 / 486.2 → 484.6 ns | 848; 7 unchanged | No detected change (p=.589/.818) |
| Sequential collection, Background/live | 723.2 → 752.2 / 780.5 → 807.0 ns | 848; 7 unchanged | No detected change (p=.167/.310) |
| Sequential string, Background/live | 462.6 → 452.8 / 518.4 → 493.8 ns | 848; 7 unchanged | No detected change (p=.394/.240) |
| Reentry, safe local control, Background/live | 1.126 → 1.106 / 1.183 → 1.170 µs | 1,576 → 1,384; 14 → 12 | No detected timing change (p=.974/.818); allocation reduction p=.002 |

The confidence intervals do not establish timing equivalence or the combined upstream-relative target. The mechanism is workspace ownership: reentry allocates one 96-byte printer for the overlapping call while sequential calls reuse the root; this workflow avoids two allocations versus an always-local safe printer. The [allocated-object source view](finalization-lane-2-evidence/scalar-repair/reentry-lease-alloc-source.txt) maps the remaining printer allocation specifically to the overlap branch. Sequential profiles contain no sampled printValue allocations; they attribute essentially all sampled objects/bytes to the pre-integration compilation admission path. Reentry printValue is 8.59% of allocated objects / 7.20% of bytes, versus 21.08% / 17.83% for the safe local control. Absolute profile object totals are not per-operation counts: benchmark calibration iterations differ.

Four existing matched CPU/heap/mutex/block captures yield [40 primary views](finalization-lane-2-evidence/scalar-repair/inspection.json) and [eight supplementary reports](finalization-lane-2-evidence/scalar-repair/extra-inspection.json), including flat/cumulative allocated and live objects and allocation source. Two supplementary source reports explicitly have no sampled printValue symbol; the wrapper initially treated pprof's no-match status as failure, then reused completed views and resumed with that result classified explicitly. No captures or timing samples were replayed for the supplementary-view wrapper correction. Frozen overlay sources map the views to their own binary.

CPU flat/cumulative profiles are dominated by runtime GC/syscalls (repaired reentry runtime.kevent 61.76% flat); this is not evidence of a network bottleneck. Jet dataPrinter.value is 6.80% cumulative and printer cleanup .57% flat in that capture; overlapping percentages are not additive. Mutex delay is runtime GC/scheduling and block delay is testing/pprof channel waits. The live samples are sparse: controls report zero sampled live heap; repaired sequential reports regexp/testing storage, and repaired reentry reports sample-weighted scavenger initialization. The latter's estimated 5,461 objects is not a literal object census. Profile-process peak RSS is 14,876,672 → 15,302,656 B sequential and 15,204,352 → 14,925,824 B reentry; these single-process samples establish neither a retention improvement nor a heap/RSS bound. Profiling uses default 512 KiB memory sampling and runs separately from timing; the older rate-1 capacity-screen caveats still apply.

The first measurement harness canceled a context between benchmark calibration calls; that failed attempt is excluded and retained. Moving context creation inside each calibration invocation repaired the harness, after which only affected binaries/stages were rebuilt. The historical output/strategy evidence above remains attributed to `56d25c8`; lane 4 must use the repaired tree plus lanes 1/3 for final combined profiles, full CI, latency/concurrency/overshoot, <=20% target and external module resolution. Local signatures are verified; GitHub verification awaits lane 4's authorized fork publication. No final acceptance or sandbox claim follows from this lane handoff.

## Runtime cost follow-up on combined 54b5614

Signed code commits `958cbed0ce48c8e47e7c2d7f8b2b4738f38b5bad` and `ec0221fb7ecd4f5416a289b8e019d60c65793872` preserve the integrated base in ancestry. The second commit repairs an active escaping regression measured in the first. Consume the complete branch, not `958cbed` alone. The fixed release corpus at the combined base failed at median ratios 1.5662 legacy, 2.1237 Background, 2.3719 live and 2.3598 restricted; the diagnostic subset below does not replace that 15-fixture text/HTML gate.

The changes address measured setup, dispatch and output boundaries:

- Runtime initializes only per-render inputs after pool cleanup; legacy execution uses the already cleared execution state. The full compilation state and full Runtime reset remain intact.
- A single list defer releases any declaration scope and restores origin/depth in the original unwind order. It removes the loop-registered defer that prevented open-coded cleanup.
- One immutable `metered` bit records that a context was supplied. The checkpoint always checks the terminal latch, then reserves through the canonical work function in that mode. Both reserveWork callers already check the latch and active context, with no intervening native call; duplicate pure guards are removed. Current Go compiler diagnostics confirm inline checkpoint/entry guards. Every cancellation opportunity and allowance remains present.
- A method-free value cannot implement Formatter/Stringer/error; its formatter dispatch avoids three unnecessary interface queries. Native hooks retain their path and checks.
- A private Set provenance bit identifies only NewSet's built-in HTML escaper. WithSafeWriter clears it. For legacy execution into a concrete owned checked bytes.Buffer destination, that built-in can write directly: neither component reenters or returns a write error. Custom escapers, active budgets, foreign/redirected destinations and transactions keep their applicable checked boundaries. No extra output pool, quota reserve or user option was added.

The only Set seam is the provenance bit and its constructor/WithSafeWriter assignments; no graph/cache/admission behavior changes. Keep lane 3's restricted zero-options/policy.execution branch and input borrow/defer-release/admission block before Runtime acquisition when integrating. The scalar workspace lease, lane 1 semantics and lane 3 compilation state/reset are preserved. The direct Runtime fixture in restricted_test.go initializes the new immutable mode.

### Causal and protection controls

Separate literal, inheritance, nested-writing, interpolation and escaping profiles precede the changes. The failing legacy literal capture attributes 23.7% of sampled CPU to full execution-state initialization; nested writing exposes runtime defer machinery. Single-factor setup/scope/dispatch overlays provide three 200ms samples each. Their role is diagnosis, not acceptance. The first selector covered literal text/HTML only; those valid cells were retained and the missing nested/interpolation cells added with a corrected selector.

Shared buffer wrappers and a separate work-validation helper were rejected after metered/live regressions. A separate legacy destination view did not justify its dispatch abstraction. The first selected implementation then showed +8.99% live escaping versus 54b5614. The guard repair preserves inline entry and removes the duplicated checks; a direct six-sample 958cbed/repair comparison improves Background/live/restricted escaping by 5.47%/5.74%/7.18% (p=.002/.009/.009). The refreshed comparison against 54b5614 detects no active-escaping timing change (p=.093/.818/.619); this is not an equivalence claim.

| Current-source validation | Evidence | Result |
| --- | --- | --- |
| Normal runtime/output/value/builtin/restricted/conformance and cancellation-observation scope | [normal](finalization-lane-2-evidence/followup-54/checks/followup-mode-normal/result.json) | Pass |
| Same focused scope with race | [race](finalization-lane-2-evidence/followup-54/checks/followup-mode-race/result.json) | Pass |
| Same scope, Darwin Go 1.16.15 | [minimum Go](finalization-lane-2-evidence/followup-54/checks/followup-mode-go116/result.json) | Pass; existing external-linker warning retained |
| Vet | [vet](finalization-lane-2-evidence/followup-54/checks/followup-mode-vet/result.json) | Pass |
| Public output/borrow/full-reset cases without the temporary measurement test | [committed tree](finalization-lane-2-evidence/followup-54/checks/followup-committed-tree/result.json) | Pass |
| Exact cancellation observations, 15 fixtures × 2 formats × 4 modes | [proof](finalization-lane-2-evidence/followup-54/terminal-repair/mode-cancellation-proof.json) | All 120 counts match 54b5614; first/middle/last cancellation controls pass |

The count probe uses the benchmark's ten warmups. Its initial cold include count included child compilation and made later warm thresholds invalid; that harness failure is retained separately and excluded. The runner was also corrected to hash every ignored injected measurement test; final protection/build receipts include it, and the committed-tree check removes that dependency.

The new public reentrant-escaper control passes unchanged 54b5614 and the fix. Two externally bounded counterfactuals establish the fast-path prerequisites: failing to clear the custom-escaper provenance bit allows `after` to append after a latched sink failure; omitting the active-meter guard lets 1,025 escaped bytes bypass the 1,024-byte limit. Both are expected failures, not production candidates. [Custom boundary](finalization-lane-2-evidence/followup-54/checks/followup-custom-boundary-negative/run.log), [meter boundary](finalization-lane-2-evidence/followup-54/checks/followup-meter-boundary-negative/run.log).

### Matched diagnostic timings

The final refresh has six interleaved one-second samples against exact 54b5614, normal Go 1.27.3 darwin/arm64, GOMAXPROCS=2, unchanged positive quotas, fixtures and ownership. Four text cases and one HTML escaping case are deliberately isolated. The earlier upstream reference is reused only with its recorded frozen binary, fixture and benchmark inputs unchanged; it was interleaved in the preceding batch, not this refresh. This does not satisfy the full fresh integrated upstream-relative acceptance program. [All samples and reference ratios](finalization-lane-2-evidence/followup-54/terminal-repair/final-analysis/timings.json), [paired intervals](finalization-lane-2-evidence/followup-54/terminal-repair/final-analysis/paired-benchstat.txt), [guard repair intervals](finalization-lane-2-evidence/followup-54/terminal-repair/final-analysis/guard-repair-benchstat.txt).

Medians, 54b5614 → ec0221f:

| Fixture/format | Legacy | Background | Live | Restricted owned |
| --- | ---: | ---: | ---: | ---: |
| Literal/text | 78.95 → 38.63 ns | 121.2 → 92.81 ns | 130.2 → 101.2 ns | 270.5 → 229.9 ns |
| Interpolation/text | 150.2 → 103.0 ns | 240.5 → 197.3 ns | 260.4 → 214.8 ns | 529.4 → 481.6 ns |
| Inheritance/text | 134.6 → 73.92 ns | 211.4 → 167.9 ns | 241.9 → 197.6 ns | 382.1 → 330.5 ns |
| Nested writing/text | 25.47 → 16.71 µs | 29.24 → 21.74 µs | 33.24 → 25.79 µs | 40.71 → 33.29 µs |
| Escape expansion/HTML | 6.508 → 4.499 µs | 8.414 → 8.297 µs | 10.33 → 10.44 µs | 12.58 → 12.63 µs |

All non-escape improvements and legacy escaping have p=.002, n=6. Active escaping has no detected change; allocation counts/B-op have no detected change. The five-case median ratios versus 54b5614 are .656 legacy, .794 Background, .816 live and .865 restricted. Against the retained upstream reference they are 1.224/2.540/2.764/3.226. Severe remaining small-template gaps stay visible: restricted interpolation is 4.087× its owned-result reference; live inheritance 3.271×, and legacy interpolation/nested-writing 1.325×/1.308×. These are diagnostic references, not the weighted release gate, and the <=20% target remains unwaived.

### Profile interpretation and remaining limits

The current source has fresh serial captures for all twenty diagnostic cells and two-worker/shared-Set captures for interpolation/text and escaping/HTML in all four modes. Each worker owns its live context. Twenty-eight unchanged 54b5614 baseline captures are reused. The selected/current pair therefore has 56 captures and 616 CPU flat/cumulative/source, allocation/live byte/object, mutex and block views. The complete intermediate 958cbed capture set remains historical evidence of the regression. [Current inspection](finalization-lane-2-evidence/followup-54/terminal-repair/final-analysis/inspection.json), [capture summary and RSS](finalization-lane-2-evidence/followup-54/terminal-repair/final-analysis/profile-summary.json), [source/receipt proof](finalization-lane-2-evidence/followup-54/source-proof.json).

Legacy built-in escaping now spends its sampled CPU in standard escaping, bytes.Buffer copying and memmove; the redundant checkedWriter dispatch is absent from that proven path. Active escaping retains its checked sink and work reservations. In current live nested writing, reserveWork is 26.72% cumulative, including cancelCtx.Err at 12.21%; checked output is 14.50% cumulative. These overlap and must not be added. Remaining costs include checked dispatch, native/interpreter entry, reflection/rangers and compilation admission. On measured Go 1.27.3, cancelCtx.Err uses an atomic load; the profiles do not show a context-mutex bottleneck.

Allocating nested workloads retain ranger/builtin construction costs. Restricted escaping retains owned buffer growth/copy costs: the current allocated-object view attributes 69.02% to growSlice and 10.12% to the owned string copy; total process object counts differ with benchmark calibration and are not per-operation counts. The normal samples remain the B/op/allocs/op evidence. Profiling uses default-equivalent 512 KiB memory sampling, separately from timing. Twenty-four of 28 current live-object views have zero sampled totals; the others contain sample-weighted runtime/setup metadata. These observations establish neither zero retention nor a goroutine/object census.

Mutex/block views chiefly show runtime allocation/scheduling, occasional sync.Pool warmup and testing/WaitGroup/profile waits. Two workers do not establish general scaling or p99 behavior. Peak RSS is distinct from live Go heap: the profiled concurrent restricted escape pair reaches 132,562,944 → 53,673,984 bytes while current sampled live heap is zero; both exhibit allocation/GC/paging work. This single-profile pair is not an RSS improvement claim or a heap bound. Existing cold/post-GC/capacity-screen limitations above remain historical, not fresh acceptance evidence.

### Integration handoff

The production code and preserved-author history are signed and locally verified. Experimental source variants are represented by exact patches against 54b5614; raw binaries/profiles remain local with hashes, commands, input/toolchain manifests, execution deadlines, bounded group cleanup and receipts. Portable copies remove host paths and other owners' queue labels. Internal plans/evidence stay fork-only and are excluded from upstream submission diffs.

Lane 4 integrates lane 3 first, then this complete branch, preserving its restricted input-admission hunk. It still owns exact combined minimum/current normal/race CI, the full representative text/HTML <=20% median gate, p50/p99/concurrency/overshoot and external fork resolution/packaging. GitHub signature verification follows its authorized normal fork push. No upstream PR/issue/tag/release was created, no review branch was rewritten, and no final acceptance or sandbox claim is made.
