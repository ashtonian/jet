# Integrated candidate: failed performance gate

The combined candidate **does not pass the <=20% performance target and is not ready for packaging or adoption**. All fixed-core workloads and exact-source compatibility CI pass, but the complete performance program remains open. The [finalization packet](finalization.md) owns the follow-up scope; consume lane 3 then lane 2 repairs before the next combined acceptance run. No upstream publication, release, tag, master merge or Karsto change is authorized by this evidence.

## Sources and correctness

Frozen production, Go harness, module and fixtures: `54b561484e1465edea963ed04579bcd24a7dbf76`. Upstream: `202464510d5b57f0afa9c5563f75babc1616682c`; correctness-only baseline: `822cf702300c6226fa56a3fcf2f8466dc6b9b712`. Later Python/report changes do not change these frozen Go inputs.

Integration preserves lane 1's private template-error marker and restricted catch/assignment semantics, lane 3's operation-local compilation state and deferred input-map clearing, and lane 2's first-cause output state and exclusive printer workspaces. The public Go-int writer/YieldBlock probe exposed overwritten borrowed bytes in the first output prototype; the same workflow passes after the workspace lease repair. Its baseline/fix evidence and original probe hash remain in [lane 2's report](finalization-lane-2.md#scalar-borrow-lifetime-repair). The authored dump patch retains its third-party attribution.

[Exact-source Linux CI](https://github.com/ashtonian/jet/actions/runs/42464957158) passes Go 1.16.15 normal, current Go normal, current Go race and the fixed-count concurrent ownership exercise. Twenty-four finalization commits verify locally and on GitHub. The module remains `github.com/CloudyKit/jet/v6`, Go 1.16 and its existing fastprinter version; fork packaging has not started.

## Fixed-core result

Go 1.27.3, Darwin arm64, Apple M2 Max; normal binaries, GOMAXPROCS=2. Six interleaved independent samples of at least one second cover every one of the original 15 fixtures in text and HTML. All 54 source/mode processes pass, with unchanged admitted inputs. Exact output conformance runs before timing. Streaming reuses its buffer; owned modes construct an independent result each render. Positive execution/parse/input budgets are unchanged in [the mode adapter](../../../execution_measurement_modes_test.go).

The acceptance statistic is the median of 30 per-fixture ratios, each formed from the six-sample median. It is not benchstat's geomean or a per-request percentile. Extended cases never alter those weights.

| Comparison | Median ratio | Overhead | <=20% |
| --- | ---: | ---: | --- |
| Candidate legacy / upstream stream | 1.5662 | 56.62% | Fail |
| Candidate legacy / correctness-only stream | 1.3953 | 39.53% | Fail |
| Metered Background / upstream stream | 2.1237 | 112.37% | Fail |
| Metered live context / upstream stream | 2.3719 | 137.19% | Fail |
| Restricted / upstream owned string | 2.3598 | 135.98% | Fail |
| Restricted / candidate owned string | 1.7414 | 74.14% | Fail |

Absolute ns/op medians for selected regressions:

| Fixture | Upstream stream | Candidate legacy | Background | Live | Upstream owned | Restricted |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Literal, text | 32.68 | 77.97 | 121.65 | 133.40 | 66.865 | 264.50 |
| Interpolation, HTML | 89.42 | 162.05 | 250.40 | 270.65 | 127.35 | 541.95 |
| Nested writing, text | 12,775 | 25,680 | 29,050.5 | 33,931 | 13,643 | 41,110 |
| Expansion escaping, HTML | 4,373.5 | 6,393.5 | 8,492 | 10,303 | 5,749.5 | 12,816.5 |
| Inheritance, text | 61.78 | 136.20 | 211.55 | 241.95 | 100.525 | 384.70 |

The literal stream has 24 B/1 allocation upstream and 0 B/0 allocations in each candidate stream despite the time regression. Nested writing changes 1,344 B/67 allocations to 1,584 B/66 in candidate streams. Restricted interpolation changes owned-upstream 160 B/4 allocations to 152 B/3. Fewer allocations alone do not establish acceptance. All per-workload time/bytes/object medians and six-sample benchstat intervals are retained in `release-54b5614/results/core-analysis.json` and `*-benchstat.txt` under the portable evidence directory below. For example, live literal text is +308.20% and live expansion HTML +135.58% (both p=.002, n=6). Some body/loop comparisons show no detected change; that is not timing equivalence and does not offset the failing small cases.

## Profile observations and limits

Eighteen separate serial/concurrent captures contain CPU, heap, mutex and block profiles. Nine rate-1 retention captures add five live-heap snapshots each: 117 files and 540 generated views. Reports include CPU flat/cumulative/call/source, allocated and live bytes/objects, allocation call/source and contention call paths. Thirteen queries have no sampled matching function. Their original nonzero pprof receipts are retained and explicitly classified; they do not mean zero cost or an incomplete capture.

The mixed legacy CPU capture assigns 5.70% flat to the no-context checkpoint fast path and 13.07% cumulative to checkedWriter.Write. Its source view maps calls through executeList and the checked destination. Live capture assigns 9.93% flat to cancelCtx.Err and 8.31% flat / 19.76% cumulative to reserveWork; source maps checkpoint/checkpointActive/reserveWork and writer byte reservations. The measured Go 1.27.3 uncanceled Err path uses an atomic load, not a mutex. These aggregate profiles also include additional data-heavy JSON/unused-input cases; their percentages do not isolate or explain the entire literal/inheritance/nested-writing gap. Follow-up owners must profile those failing core paths separately.

The former warm compilation-map constructors leave the sampled allocation tables after compact admission. Remaining metered allocations include newScope, executeYieldBlock, dataJSONEncoder traversal/string encoding and reflection. Restricted captures add independent mutable-input copies and result storage; frozen source maps reflection map creation/assignment to input cloning. Buffer growth and the result String copy remain ownership costs, also present in the matched owned baseline. The [lane 3 strategy ledger](lane-3-finalization.md#strategies-accepted-and-rejected) and [lane 2 buffer screens](finalization-lane-2.md#historical-buffer-strategy-screen-56d25c8) retain accepted/rejected variants and their separate costs. No new buffer reserve/pool or weaker validation is selected here.

Concurrent CPU includes runtime.usleep/madvise/condition waits alongside buffer, escape and work paths. Mutex delay is chiefly runtime allocation/unlock paths; block delay mainly comprises testing channels and worker joins, with a small pool-pin contribution. This is not a concurrency-scaling or load-generator-capacity verdict. The normal fixed-worker sweep and capacity control remain pending.

Default-sampled post-benchmark live heap is sparse and largely runtime metadata. Separate rate-1 snapshots deliberately keep the Set, template and 256 KiB input alive. After large-then-small plus GC, owned upstream/candidate/restricted observations are approximately 275/270/273 KiB, dominated by that live input. Corresponding uninstrumented process maxima are 12/12.53/12.53 MiB RSS; Darwin getrusage reports bytes. Streaming buffer retention is an explicit different ownership contract. This finite scenario and per-entry scratch caps prove no global heap/RSS bound. No trace was needed to make these limited observations; no scalability or scheduling conclusion is inferred from unexplained residual runtime costs.

## Cancellation observations

Each native case has 128 synchronized observations and separate deterministic cancellation controls. Reported p99 is sorted[(n-1)*99//100], an observed order statistic, not a confidence bound. The timing context itself is instrumented; these observations are separate from normal throughput and render-tail measurements.

| Operation | Canceled / total | Median overshoot, ns | p99, ns | Maximum, ns |
| --- | ---: | ---: | ---: | ---: |
| Interpreter loop | 128/128 | 1,229.5 | 7,083 | 9,041 |
| Prepaid case conversion | 128/128 | 152,916 | 175,625 | 185,292 |
| Compilation, canceled subset | 53/128 | 1,584 | 57,709 | 60,959 |
| Input admission | 128/128 | 1,125 | 24,083 | 74,709 |

Compilation has 73 negative overshoots: return preceded the request timestamp. It also has two uncanceled returns with nonnegative overshoot. The earlier `cancellation-summary.json` incorrectly classified all 75 uncanceled returns as completed-before-request; `cancellation-distributions.json` corrects the classification without replaying the observations. The request timestamp precedes cancel(); completion can race between the final checkpoint and return. A nonnegative overshoot with nil error alone proves no missed observable cancellation. All-sample compilation median remains -250 ns, not negative cancellation latency. No measured context checks continue after synchronous return.

The cooperative callback has median/p99/max 1,604.5/5,333/8,583 ns across 128 observations. The intentionally blocked callback is explicitly released and joined, with 1,520.5/10,875/54,917 ns. These are controlled release observations, not bounds on arbitrary native callbacks. Restricted mode rejects the supplied capability without invoking it. No wall-clock assertion or dedicated fuzzing was added.

## Reproduction and evidence

Portable evidence is under [release-54b5614](finalization-lane-4-evidence/release-54b5614/export.json). It includes commands, input/config/toolchain manifests, normal samples/intervals, all generated profile views, CI/signature proof, cancellation records and bounded result receipts. Repeated input maps are interned by hash. Host paths are replaced, host-gate queue/holder notices omitted and text whitespace normalized; original and portable hashes are separate. Original binaries/pprof remain local under `.git/finalization-evidence/release-54b5614`, with hashes in the portable packet. This report does not claim that those binary artifacts are published.

From the recorded source, set ARTIFACTS and RECEIPTS to new local artifact/receipt directories; do not overwrite frozen evidence:

```sh
python3 docs/plans/template-execution-budgets/finalization-lane-4-evidence/run_matrix.py "$ARTIFACTS" prepare 54b561484e1465edea963ed04579bcd24a7dbf76
python3 docs/plans/template-execution-budgets/finalization-lane-4-evidence/run_stage.py "$RECEIPTS" build run-exclusive 600 python3 docs/plans/template-execution-budgets/finalization-lane-4-evidence/run_matrix.py "$ARTIFACTS" build
python3 docs/plans/template-execution-budgets/finalization-lane-4-evidence/run_stage.py "$RECEIPTS" core run-exclusive 5400 python3 docs/plans/template-execution-budgets/finalization-lane-4-evidence/run_matrix.py "$ARTIFACTS" core
python3 docs/plans/template-execution-budgets/finalization-lane-4-evidence/run_stage.py "$RECEIPTS" analysis run-check 240 python3 docs/plans/template-execution-budgets/finalization-lane-4-evidence/analyze_matrix.py "$ARTIFACTS" --partial
```

`--partial` is diagnostic only: final analysis rejects absent extended/phase/latency/concurrency cells and missing metrics. Expected full coverage is 136 extended, 108 phase, 406 latency and 720 concurrent cells, six samples each; the inherited named-map panic has no valid baseline timing and remains an explicit failing control. Final concurrency also requires all six harness-capacity controls. Worker-owned live contexts/vars/results and exact positive budgets are preserved. Driver lifecycle controls verify source/env/binary/artifact invalidation, deep-copied receipt inputs, TERM/INT child cleanup and rejection of incomplete coverage. Installed host-gate owns every admitted stage with execution deadlines, bounded cleanup and queue time recorded separately. Full suites use CI; no local full-suite replay is attributed to this report.

## Remaining gates

- Structural follow-up fixes and their controlled failing-baseline/passing-fix evidence; preserve all protection and supported behavior.
- Repaired combined candidate: exact minimum/current normal/race CI and six-sample full core, extended, phase, p50/p99 and fixed-worker/capacity program. Current extended/phase/latency/concurrency timings are unexecuted, not passing.
- Only after generic correctness and performance pass: separate fork module/self-import packaging, packaged exact-source CI and a fresh external consumer resolving the pushed immutable revision without replace or duplicate Jet.
- Verified signatures for every later commit and accurate existing fork PR descriptions. No performance waiver, sandbox claim, tenant adoption or #209 closure follows from this failed candidate.
