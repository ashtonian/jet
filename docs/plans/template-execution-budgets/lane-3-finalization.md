# Lane 3 finalization: compilation and input admission

Baseline: `0e277b3017b4afae1f984663c4e9244b4de0b290`. Implementation: `7f7f38a` on `work/finalization-lane-3`. The [finalization packet](https://github.com/ashtonian/jet/blob/f1466f7cb549fc453fb38f3dbdae7cde8adce609/docs/plans/template-execution-budgets/finalization.md) and [review](https://github.com/ashtonian/jet/blob/f1466f7cb549fc453fb38f3dbdae7cde8adce609/docs/plans/template-execution-budgets/final-review.md) supersede the earlier acceptance report. This handoff covers the compilation/input component; combined release acceptance remains open.

## Change and preservation

Compilation admission uses four inline source/version slots, spilling to operation-local indexes for larger graphs. Runtime owns its initial admission storage; standalone compiler operations reuse a reset admission object. Cleanup clears every context/name/template reference and drops all dynamically grown slices/maps before reuse. Runtime's existing execution reset clears its inline state. No request state is stored on Set, AST or cache.

Each compiled template retains one transitive depth integer. It certifies successful acyclic compilation and checks the longest dependency path before a repeated visit is skipped. Unlimited standalone cache lookup uses that certificate plus originating-Set and namespace checks. Finite callers re-account unique source names, maximum observed source sizes and distinct AST versions. Cold parses still walk cached dependencies against every active canonical ancestor, including previously visited subgraphs. Development reparsing continues to charge actual construction and source growth; shrinking a source does not refund its earlier charge.

Input admission uses eight active-path identities and depth-bounded overflow, replacing a per-operation active map. Immutable scalar values can retain their existing unaddressable representation; addressable caller cells are copied. Mutable containers remain independent snapshots, and unused forbidden inputs/cycles are still rejected. Empty scopes allocate no vars map. Only the closed restricted render entry loans a cleared vars map for at most 16 initial entries; larger scopes use fresh storage. Constructor globals and LookupGlobal snapshots escape their calls and use fresh maps. Logical item/name/payload charges remain unchanged by storage reuse. Returned strings and mutable data graphs are never recycled.

Shared-file integration hunks are limited to `executionState.compilationState` and `startCompilation` in execution.go, and the restricted input-admission block in exec.go. Preserve those additions when integrating Lane 2's output setup/recovery changes. Existing `st.execution = executionState{}` clears the added inline field.

## Controlled observations

All samples use normal Go 1.27.3 darwin/arm64, GOMAXPROCS=2 and the review's exact positive budgets. Streaming and newly owned string modes remain separate. Text and HTML cases are both retained. Six interleaved one-second samples establish the focused paired comparisons; capacity/input-shape screening uses three shorter samples and is not release acceptance. Source/overlay hashes, per-stage commands, execution deadlines and results are in [the evidence index](lane-3-finalization-evidence/index.md).

| Same-workflow comparison | Baseline | Graph prototype | Scope |
| --- | ---: | ---: | --- |
| HTML interpolation, metered live | 573.8 ns; 1,000 B; 11 allocations | 301.2 ns; 152 B; 4 allocations | Six matched samples, compilation change isolated; input implementation unchanged |
| HTML interpolation, metered Background | 552.5 ns; 1,000 B; 11 allocations | 285.1 ns; 152 B; 4 allocations | Same batch, separate mode |
| Cached layered diamond lookup | 4,708.5 ns | 1,639.0 ns | Same compact state with repeated traversal enabled/disabled, six one-second samples |
| HTML interpolation, restricted owned result | 661.6 ns; 720 B; 9 allocations | 586.6 ns; 320 B; 7 allocations | Later matched batch: unpooled scope/fixed 64-slot input path versus selected scope reuse/8-slot path |
| HTML include, restricted owned result | 991.8 ns; 816 B; 14 allocations | 850.7 ns; 416 B; 12 allocations | Same later batch |

These are distinct controlled batches, not a synthesized end-to-end ratio or the full 30-case performance verdict. The first graph snapshot predates one additional per-dependency checkpoint during cold metadata construction. The later batch includes that checkpoint and matches selected production code, apart from an explanatory comment and added tests. Passing prior cold measurements are not silently relabeled as measurements of that final change.

The first batch's flat allocation profile attributes 65.13% to newParseAdmission/prepareSource/admitCached; those sites leave the sampled allocation table after the graph change. The remaining input scope map accounts for 32.30% in the intermediate profile. Source views identify map creation and insertion, motivating the restricted-only loan. In the selected profile, those scope allocations also leave the sampled table. Remaining allocators include result ownership, printing, output wrappers and scope/recovery. CPU flat/cumulative, alloc_space, inuse_space, mutex and block views were read separately. Serial channel waits belong to the benchmark harness; they do not prove concurrency scalability. RSS and sampled live heap are retained separately from cumulative allocation; no memory cap is inferred.

### Strategies accepted and rejected

- Four graph slots are selected. Lazy spill with zero inline slots retains common-case allocations. Eight/sixteen remove allocations for medium graphs but regress the small cached render screens. Runtime sizes are 552/712/1,032 bytes for 4/8/16 slots, versus 208 bytes in the original candidate. The size/reset tradeoff is material, even though allocations fall.
- Thresholds 3/4/5, 7/8/9 and 15/16/17, plus 1/64 templates, are screened. Four-slot named lookup at 1/4 templates uses 48 B/3 allocations versus baseline 896 B/10. At 64 templates it uses about 16.8 KiB/33 allocations versus baseline 12.0 KiB/28: the spill representation improves measured time but allocates more. This limitation remains explicit.
- The immutable summary is one maximum-depth integer per template: linear retained metadata, no transitive source-closure copies. A repeated-walk control isolates the benefit of memoizing visits. Summing child costs would double-charge diamonds; unique name/version accounting remains operation-local.
- The reset standalone admission pool removes a surviving warm-lookup allocation: the no-pool control uses 400 B/4 allocations versus 48 B/3 at one template. Cold/post-GC correctness does not depend on pool hits; large scratch capacities are dropped.
- A 64-entry input identity array avoids hash bookkeeping but enlarges the stack frame. The selected eight-entry path with overflow improves the matched small-render result while preserving deeper accepted inputs. It retains the same 64-depth ceiling. Input-shape screens and profiles cover the deeper-copy tradeoff separately.
- Vars-map reuse is restricted to closed renders and at most 16 initial entries. Clearing/failure/tenant isolation, mutable snapshots, large-then-small traffic, post-GC and concurrent reuse have focused controls. Pool entry limits are not a global retained-memory bound.

### Selected input-shape cost and profile limits

Three 200 ms screening samples compare the same scope-reuse implementation with a fixed 64-slot path and the selected eight-slot path/overflow. These are diagnostic medians, not final tail-latency acceptance.

| Input shape | Fixed 64 slots | Selected 8 + overflow | Selected bytes / allocations |
| --- | ---: | ---: | ---: |
| Empty | 308 ns | 274 ns | 202 B / 5 |
| Scalar | 316 ns | 287 ns | 202 B / 5 |
| Nested | 3,866 ns | 4,308 ns | 5,098 B / 68 (adds 288 B / 3 versus fixed 64) |
| Shared children | 1,650 ns | 1,593 ns | 1,170 B / 21 |
| Large map | 40,525 ns | 38,122 ns | 23,078 B / 649 |

The nested-case regression is approximately 11%; it is not hidden by an aggregate. Input state shrinks from 2,640 to 424 bytes, at the cost of depth-dependent overflow allocations. In the selected nested/large profile, clone is 6.51% CPU flat / 70.70% cumulative; reflection assignment, allocation and type/method inspection remain material. The alloc_space view attributes 47.57% to reflect map assignment, 19.42% to reflect map creation, 15.54% to reflect allocation and 3.89% to active-path overflow. The alloc_objects view separately attributes 55.36% of estimated objects to reflect allocation and 2.36% to overflow. Frozen-source views locate overflow in enterContainer and container copies/assignment in clone. Those costs remain for the final combined workload assessment; no data validation or snapshot copy is dropped to remove them.

The selected shape run has zero sampled inuse_space/inuse_objects at profile capture, not zero retained memory. The retained-cache probe separately keeps a 64-template Set alive: selected live-heap delta is about 51 KiB and process maximum RSS about 8.2 MiB in that short process. The large difference between cumulative allocations and the retained/RSS observations is preserved. Pool/control allocations can be missed by sampling; “no matches” means no sampled attribution, never zero cost. Serial mutex/block samples mainly show runtime/harness activity and cannot establish concurrent contention behavior. Functional concurrent vars-map reuse passes the race detector; final combined concurrent performance profiles remain mandatory in Lane 4.

Source reports use the module-path trim only to resolve unique frozen paths under `.git/finalization/*-source` and `followup-frozen`, not current production files. Hash proofs tie each listed source to the exact overlay/build manifest. Additional alloc_objects/inuse_objects views reuse the original heap profiles; they are not new performance samples.

## Validation and remaining gates

The original cached diamond fails its finite-work assertion; the fixed workflow passes. A prototype initially skipped a cached subgraph that conflicted with a newly active source version; its failing control is retained and the final normal/race checks reject it. Other regressions cover all-path depth, cold/warm policy tightening, spills, development source growth/shrink, unique source count, distinct version construction, native panic/reader ownership, cache/namespace isolation, cancellation and immutable snapshots. Runtime scratch is zero after a public render, including dynamically grown graph state. Dedicated fuzzing was not added or run.

Selected normal/race named-package checks and vet pass under installed host-gate. Benchmark stages use exclusive admission; test/analyzer stages use their designated focused/check lanes. Every execution has a deadline and bounded TERM/KILL cleanup, distinct from queue wait. The signed implementation commit is verified locally. GitHub verification is pending Lane 4's authorized fork publication of the integrated ancestry; this lane creates no PR, upstream write or force push.

The full <=20% median target is **not accepted** by these component measurements. In particular, the remaining small-template cost is substantial relative to unchanged upstream. Lane 4 owns exact combined minimum/current-Go normal/race CI, the complete fixture/mode/latency/concurrency/overshoot matrix and external fork-module resolution. The overall candidate is not release-ready from this handoff. Karsto adoption remains excluded.
