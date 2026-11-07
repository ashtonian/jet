# Lane 4 execution evidence

Execution authorized by the owner on 2026-11-07. Tenant adoption, fork packaging and upstream submission remain gated.

## Inputs and handoff

- Runtime: upstream `202464510d5b57f0afa9c5563f75babc1616682c`, unchanged on this lane.
- Neutral corpus/harness: `f055df683755104e9d94e2fc3ef5c6b0b937b004`; local signature and GitHub `verified=true`, reason `valid`.
- Corpus: `testData/execution_workloads.json`; 15 fixtures, text/HTML variants and compile/cached-execute/parse-execute stages.
- Go `go1.27.3`, darwin/arm64; benchmark `-cpu=2`, normal build, three independent 100 ms samples per case.
- Dependency: Jet's existing fastprinter `v0.0.0-20200109182630-33d98a066a53`. Karsto's MVS graph selects fastprinter `v0.0.0-20251202014920-1725d2651bd4`; do not attribute this difference to metering.
- Owned additions: `execution_workloads_test.go`, JSON corpus, compatibility workflow and this lane's evidence. No peer runtime/evaluator code was modified or integrated.

The fixture commit contains only generic tests/data. Keep the workflow and internal execution documents out of a focused upstream fixture submission. Packaging is a later isolated commit after all protection contracts pass.

## Baseline observations

`TestExecutionWorkloads` passes in text and HTML. The block/content fixture deliberately preserves Jet's automatic block execution (`[][value]`). Compile stages construct a fresh Set and dependency graph; cached execution reuses a parsed template and reset output buffer. Parse+execute constructs a fresh output buffer. These measure different allocation lifetimes.

Median of three **ns/op samples**, not per-request latency percentiles:

| Cached Execute fixture | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| Interpolated header, text | 78.87 | 24 | 1 |
| 8 KiB body, text | 297.7 | 24 | 1 |
| 64 KiB body, text | 2029 | 26 | 1 |
| 64 × 64 silent loop, text | 56884 | 2624 | 131 |
| Expansion-heavy escaping, HTML | 4592 | 24 | 1 |
| Failed try rollback, text | 890.7 | 584 | 11 |

The separate profile run samples cached text execution of the 8 KiB body, silent loop and expansion fixture, 2 seconds per workload. CPU flat/cumulative, allocation/live heap, mutex and block profiles were inspected. In that mixed run, `executeList` accounts for 81.47% cumulative CPU, `memmove` 24.63% flat CPU, and printing 36.44% cumulative CPU. `Runtime.recover` accounts for 89.72% of sampled allocation bytes; this is cumulative allocation traffic, not retained heap or RSS. The post-GC live heap sample contains 513 KiB attributed to runtime thread allocation. Mutex samples are runtime GC synchronization; block samples are primarily benchmark-runner channel waits. This sequential profile supplies no concurrent contention verdict and no metering overhead comparison.

Separate latency sampling and concurrent execution fixtures are in `execution_measurements_test.go`. Latency sampling
retains at most 100,000 clock observations after 100 warmup renders; its timing overhead is separate from the throughput
benchmark. Three independent 100 ms samples, normal build, `-cpu=2`:

| Latency fixture | Median sampled p50, ns | Median sampled p99, ns |
| --- | ---: | ---: |
| Interpolated header, text | 83 | 166 |
| 8 KiB body, text | 292 | 417 |
| Silent loop, text | 56750 | 73250 |
| Expansion-heavy escaping, HTML | 4375 | 6209 |

The concurrent harness runs four representative fixtures in text/HTML with exactly 1, 2 and 8 workers, shared templates
and separate Sets. With `GOMAXPROCS=2`, the host-fit worker count is already the 2-worker case. Each worker owns its output
buffer, input maps remain read-only and all workers join before return. Normal three-sample runs and a separate
`-race -benchtime=32x` exercise passed. Concurrent ns/op is inverse aggregate throughput, not per-request latency. No
scalability or contention verdict is claimed without concurrent profiles. Logs are `latency-concurrency.log` (SHA-256
`932684b6a285d548d5e203130b65e11d86b2598d5b8649dc0e11e90cad1f0899`) and `concurrent-race.log` (SHA-256
`413628b83d168d7b73dcc91ec811af875f52d26eeb47c6e6e7ddab51bf1ea278`). Source stayed unchanged during these stages; the
latency manifest was captured after execution, as its receipt records.

## Reproduction and receipts

From the fixture commit, with the installed host gate and bounded execution:

```sh
GATE_LOCK_COMMAND_TIMEOUT=120 GATE_LOCK_COMMAND_KILL_AFTER=5 gate-lock run-focused jet-workloads -- go test -count=1 -timeout=60s -run=TestExecutionWorkloads .
GATE_LOCK_COMMAND_TIMEOUT=300 GATE_LOCK_COMMAND_KILL_AFTER=5 gate-lock run-exclusive jet-baseline -- go test -run='^$' -bench=BenchmarkExecutionWorkloads -benchmem -benchtime=100ms -count=3 -cpu=2 -timeout=240s .
```

Profiles use the same benchmark with `-bench='BenchmarkExecutionWorkloads/(nested_silent|body_8192|escape_expansion)/text/cached_execute$' -benchtime=2s -cpu=2`, plus `-cpuprofile`, `-memprofile`, `-mutexprofile`, `-blockprofile`, and an explicit output binary path. Inspect CPU flat/cumulative, heap `alloc_space` and `inuse_space`, mutex and block reports separately.

Durable artifacts in this checkout: `.git/lane-4-evidence/`; `measurement-inputs.json` records tracked/untracked source hashes, `measurement-receipt.json` records source HEAD/toolchain/environment and outer 360-second execution/5-second cleanup bounds, and stage receipts record commands/results. Queue time is separate. Benchmark and profile stages both exited zero; actionlint also passed for the compatibility workflow.

| Artifact | SHA-256 |
| --- | --- |
| `bench.log` | `7c922fbf7082458336110829127c7dfaa77d1b3a9b388c84d41abc0fde827e7a` |
| `profile.log` | `d01b3bee1926990ec0308ef80bbcc2a57c50d9e38c4460e1c5567fb72fd9e62e` |
| `measurement-receipt.json` | `343b00627d825f17f673fde283aa3a4cf0dec340ec3401b1407abcc0e3d5aaca` |
| `cpu.pprof` | `998bcb77b58abbf66feb34378410e0e16cdff81193087e95fed4224e5747822f` |
| `heap.pprof` | `d2ff0a0b5300e104aa6870a48bf53a069a33b0f217bc7240a273d0a0f7f8ec0b` |
| `mutex.pprof` | `78a000491d1211d96cc700e67276e5ff4cfe506aed0884e716e7a3f341ff507a` |
| `block.pprof` | `66db40a8eca49b9acec72f97351cd198fd90e6e0ca43261ae7b74b2f9cb95d43` |

## Remaining acceptance

The compatibility workflow runs full normal suites on Go 1.16.15 and stable, plus a separate stable race suite. All three
passed at `accce48cdc3d7813eba528c8b90d10697e202f02` ([run](https://github.com/ashtonian/jet/actions/runs/42424833224)).
The additional measurement harness needs that matrix on its own pushed SHA. Action pins were checked against the official
[checkout v7.0.1](https://github.com/actions/checkout/releases/tag/v7.0.1) and
[setup-go v7.0.0](https://github.com/actions/setup-go/releases/tag/v7.0.0) release notes; both were current stable releases.

Lane 1 now supplies the explicit foundation `77025e5a5bbef3d6dd1208c49d16005e77250d77`, production `d2f089c`, in
[PR #3](https://github.com/ashtonian/jet/pull/3). Its checkpoint/reserve/terminal/call/load/transaction contracts were
reviewed without modifying or consuming the evaluator. Lane 2's checked-size/helper preparation is at `bea4624`; no
accepted J04/J05 protection handoff is available yet. Lane 3's parser/input preparation was fetched through `9432a63`;
complete restricted invocation/builtin/namespace acceptance remains pending. These are independent preparation results,
not a combined candidate.

No final fork legacy/metered comparison, integrated Karsto comparison, candidate latency/concurrency sweep, sustained
CPU/RSS or cancellation-overshoot acceptance is complete. No runtime/protection matrix row is certified here. F01/K02
require explicit accepted lane 1/2/3 handoff SHAs and the complete restricted contract, followed by I01–I08. No release,
merge, #209 closure or upstream submission has occurred. Canonical preparation PR: [#4](https://github.com/ashtonian/jet/pull/4).
