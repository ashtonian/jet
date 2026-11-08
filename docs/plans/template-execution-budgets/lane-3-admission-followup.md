# Lane 3 admission follow-up

Implementation: signed `7001b1e`. Base: combined `54b561484e1465edea963ed04579bcd24a7dbf76`. The [finalization packet](finalization.md) retains the failed <=20% release gate. This component changes input admission and the restricted default-options boundary; compilation graph representation and runtime/output setup are unchanged.

## Structural changes and preservation

Scalar cloning returns before container dispatch and deferred active-path cleanup. Type inspection still denies every methodful value. For supported scalar/container types, pointer-receiver inspection is needed only for package-defined types; unnamed/predeclared scalar/container types cannot acquire those methods. Interface method sets are checked directly. Structs remain denied regardless of promoted methods. Named scalars, containers and map keys keep the reflective pointer-method check. No type cache or persistent metadata is introduced.

A `map[string]interface{}` snapshot uses native iteration/insertion after the shared map/key capability checks, active-path registration and item/storage reservations. Each entry retains its key-byte charge and static interface layer's step/depth checkpoint, including nil entries. Dynamic children use the same recursive clone implementation. Named maps and other supported representations retain the generic reflection path. Every mutable occurrence is still copied independently; aliases outside the active path are not memoized. Cycles and unused capabilities still fail closed.

The closed restricted entry's zero ExecutionOptions selects the constructor-validated immutable policy directly. Nonzero options still use the existing intersection with the policy. Context validation, snapshot admission, compilation admission and every runtime checkpoint/charge remain in place. Shared-file changes are confined to this conditional in exec.go's restricted input block; retain Lane 2's runtime/output changes separately.

There is no new pool, global cache or retained graph summary. Input identity storage, vars-map retention thresholds and graph certificates keep their previous bounds/tradeoffs. Native map copies remain execution-owned allocations; lower cumulative allocation is not a live-heap/RSS cap.

## Causal observations

The failed combined candidate's aggregate restricted profile mixes core fixtures with 256-item data/JSON workloads. It is not used to explain small scalar costs. Separate three-second normal-build profiles use the existing mode harness, Go 1.27.3 darwin/arm64 and its exact positive protection budgets. Empty/header, scalar/header and named-map fixtures are profiled independently.

The scalar profile attributes 20.60% cumulative CPU to admitInputScopesInto: map iteration/start accounts for 21.95% of that admission sample, clone 19.51%, byte reservation 17.07%, item reservation 12.20% and insertion 10.98%. Clone's call view includes pointer-type inspection and defer machinery. In the named-map profile, clone is 14.50% cumulative CPU; reflective insertion, iteration, creation and key/value extraction dominate its callees. The alloc_space view attributes 54.14% to reflective map insertion, 9.62% to reflective value allocation and 8.91% to reflective map creation. The scalar allocation profile instead attributes sampled allocation to owned result/sink storage, with no sampled input allocation. Empty admission is 11.05% cumulative CPU; initial graph admission is 10.24%. These are sampling observations, not time ratios.

All fixed 15 fixtures use nil Data and empty or scalar-string vars; native map-copy results therefore do not contribute a direct gain to that acceptance corpus. Short three-sample 200 ms screens isolate the scalar/container split plus method proof from the additional direct-map copy. They are diagnostic only. Six interleaved one-second samples per cell compare the exact frozen combined source with the selected implementation, under identical Go/toolchain/budgets/GOMAXPROCS=2. Timing samples have no profile instrumentation. This follow-up measures selected HTML workflows and additional input shapes; it is not the full text/HTML acceptance corpus.

| Restricted owned HTML workflow | Combined baseline ns | Selected ns | Selected / baseline |
| --- | ---: | ---: | ---: |
| Literal header | 261.45 | 245.30 | 0.9382 |
| Interpolated header | 565.85 | 519.30 | 0.9177 |
| Include | 838.85 | 784.10 | 0.9347 |
| Inheritance | 387.60 | 368.55 | 0.9509 |

Those four workflows keep the same bytes/allocations per operation. The additional named-map interpolation fallback is 825.65 -> 843.90 ns (1.0221), with unchanged 533 B/9 allocations. Its slower median is retained, not averaged into the native-map improvement; this batch does not establish its cause. Named types keep generic reflection and pointer-method checks.

| Additional data shape, literal result | Baseline ns | Selected ns | B/op, baseline -> selected | Allocations, baseline -> selected |
| --- | ---: | ---: | ---: | ---: |
| Empty | 262.05 | 237.35 | 130 -> 130 | 3 -> 3 |
| Scalar | 284.00 | 260.30 | 130 -> 130 | 3 -> 3 |
| Small native map | 566.85 | 422.80 | 514 -> 466 | 8 -> 5 |
| Nested maps | 4,340.0 | 2,881.0 | 5,026 -> 4,450 | 66 -> 30 |
| Shared children | 1,689.0 | 1,246.5 | 1,098 -> 1,002 | 19 -> 13 |
| 128-entry map | 40,778.5 | 23,244.0 | 23,007 -> 16,861 | 647 -> 263 |

These data snapshots are still copied even when unused by the template. No improvement to the full fixed-fixture median is inferred from the subset or from earlier upstream timings.

### Selected profiles and retained memory

All CPU flat/cumulative, source/call, alloc_space/alloc_objects, inuse_space/inuse_objects, mutex/block and RSS views were read separately. Source reports resolve immutable `.git/admission-followup/*-frozen` overlay paths and `defaults-exec.go.txt`. Their hashes and the production proof are retained in [the evidence index](lane-3-admission-followup-evidence/index.md). A later ordinary comment narrows the method-proof wording to supported types; executable source is unchanged. The deterministic cancellation test was added after timing sources were frozen and passes in the final normal/race batch.

In the separate selected scalar profile, admission is 16.00% cumulative CPU versus 19.80% in the newly profiled baseline; clone is 3.20% versus 5.94%. Clone's selected call view has no sampled PtrTo/defer machinery, while the baseline view attributes samples to both. Frozen execute source shows the zero-default policy selection replacing constrainExecution on that path. These fractions describe sampled attribution; the paired timing table measures the outcome.

For the native small map, admission is 30.98% cumulative CPU versus 36.32%. Baseline clone's callees include SetMapIndex, MapIter.Next/Key/Value and MakeMapWithSize. Selected container copying instead calls native map iteration/insertion/allocation. Allocation source views locate the remaining two map-storage allocations at make and insertion; reflective key/value cell allocation disappears from that path. Deep/shared/large profiles keep reservation, active-path work, slice copying and map storage visible. Baseline clone is 64.91% cumulative there; selected clone/container remain about 45%. Profile totals contain different adaptive iteration counts and are not compared as equal-work byte totals; B/op and allocations/op above provide the matched comparison.

Scalar in-use profiles have no sampled live allocations at capture. Other serial snapshots primarily retain sampled runtime/harness storage. Twelve-worker interpolation profiles retain about 2.5 MiB/1,281 sampled live runtime objects in baseline and 3.0 MiB/1,318 in selected; these are not retained input-graph measurements. Maximum RSS is 15,745,024 -> 15,925,248 bytes for scalar, 14,729,216 -> 14,827,520 for small native map, 15,810,560 -> 15,482,880 for the deep/shared/large process, and 29,671,424 -> 29,868,032 for the concurrent process. Single process maxima and sampled heap sizes establish no memory cap or causal retention reduction.

Serial mutex samples are runtime activity, and block samples are benchmark channel waits. Concurrent mutex delay is predominantly runtime unlock (about 96%); the application sync.Mutex contribution resolves to sync.Pool.pinSlow across existing runtime/input/printer pools. Block views similarly show harness channel/WaitGroup waits and pool initialization. No cancellation-context mutex is attributed; Go 1.27.3 cancelCtx.Err uses an atomic load. Shared-Set and independent-Set interpolation cases use independently created worker contexts. Their profiled single-run timings are diagnostic, not throughput/tail acceptance. The harness excludes data_unused_256 from its concurrent fixture list, so that selector added no concurrent data-copy case; full data-heavy concurrency remains Lane 4's combined gate. No dedicated contention/scalability claim is made.

## Validation and remaining gates

Focused normal and race package checks pass for input admission, restricted workflows, compilation/graph/bundle controls, execution-mode conformance and measurement bounds; go vet passes. New controls cover exact byte/step/depth thresholds for string/nil map entries on native and named representations, pointer methods on scalar/map/slice/key/nested values, and deterministic cancellation at every map admission checkpoint. Existing mutable-copy, addressable-cell, cycles/sharing, long-path overflow, tenant isolation, post-GC and concurrent controls remain included.

Validation uses installed gate-lock with 180-second execution and five-second TERM cleanup; Go tests have 90-second deadlines. Benchmark stages use exclusive admission with their own child deadlines/cleanup. Queue waiting is recorded separately. No dedicated fuzzing or full local suite is run. Full minimum/current-Go CI and final performance acceptance remain Lane 4's combined-candidate gates. No PR, upstream write, packaging change or release is made by this component.

The component's executable changes and focused evidence are validated for integration. The failed <=20% combined target is not accepted or waived. Lane 4 still owns the final exact combined candidate, full fixed-corpus/text/HTML, tail/cancellation/concurrency program, minimum/current-Go CI and external module packaging. Publication/signature verification on GitHub is pending that authorized fork integration; local signing is verified.
