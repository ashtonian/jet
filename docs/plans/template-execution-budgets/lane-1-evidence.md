# Runtime foundation evidence

Execution authorized 2026-11-07. This lane delivers generic runtime ownership and interpreter metering; tenant acceptance remains dependent on values/output, compilation/capabilities and Karsto integration.

## Immutable inputs and handoff

- Upstream v6.3.3/refreshed master: `202464510d5b57f0afa9c5563f75babc1616682c`; planning base `b0bbc0761fe98bbee4b6c74f7df74454f4c2d333`.
- Canonical code: `d2f089c` on `work/lane-1-runtime`; correctness `3863ed9`, context/fuel/depth `86fce7a`, public YieldBlock correction `388f288`, scan accounting/sink adapter consolidation `d2f089c`.
- Final canonical code/tests/documentation handoff: `77025e5`; production remains `d2f089c`. [Upstream context/fuel/depth draft PR #238](https://github.com/CloudyKit/jet/pull/238) maps it to `1573061999cc3e9fdc543d5635e762d364e199ff`, stacked on #237. Submission content was checked against canonical Go/dependency/template blobs; internal plans/RFCs and packaging are excluded.
- [Fork handoff PR #3](https://github.com/ashtonian/jet/pull/3). [Upstream correctness PR #237](https://github.com/CloudyKit/jet/pull/237) uses generic submission commits `8b5e7b0` and `822cf70`, excluding all planning documents and packaging.
- Go 1.27.3, darwin/arm64; Go 1.16.15 compatibility uses the official archive with SHA-256 `28365c5c252970c10f2627dc0aa63a2ec3df04a92df4d45ed83dbe464732c3e0`. Module/dependency files are unchanged.
- Every created commit was GPG-signed with the effective repository identity, locally verified and checked through GitHub's verification API. Source changes preserve upstream authorship/history.
- Peer review: lane 2 preparation through `bea4624` (checked sizes/cost inventory) and lane 3 through `bd2479e` (authorship-preserving #236 adoption, synchronous lexer, bounded compile API and input snapshots), and lane 4 through `accce48` (compatibility/workload harness) were fetched/reviewed. Their dependent changes are not silently included in this lane or assumed present on master.

Durable local artifacts live under `.git/lane1-validation/` in this checkout. JSON receipts record commands, source hashes including untracked fixtures, toolchain/configuration and status. Final source proof matches the Go/dependency/template inputs of the passing final stages to `d2f089c`. Logs use the stage names below. No validator is intentionally left running after delivery.

## Controlled baselines and fixes

| Case | Controlled baseline | Fixed result / evidence |
| --- | --- | --- |
| Sink errors | `j01-baseline`: text errors lose errors.Is identity; escaped/raw/HTML/JSON output and try commit report success | `j01-writer-fix`, `j01-fix`, final suites preserve original causes and io.ErrShortWrite |
| Runtime retention | Same baseline retains writer/Set after success, ordinary error and non-error panic | Field assertions plus ranger reset tests pass; cleanup occurs before pool publication |
| Try context | Failed range leaves catch in child data context | Catch renders root context; nested rollback still produces only outer fallback |
| Self yield | `recursion-baseline-actual`: original source fatally overflows with `debug.SetMaxStack(256<<10)` in a separately supervised child | Same source/cap returns depth termination; subprocess composition regressions cover self/mutual yield, Runtime.YieldBlock, include, exec and includeIfExists |
| Silent work | `silent-baseline`: original finite 100,000-advance source ignores cancelled caller context | Public ExecuteContext rejects cancellation and fuel exhaustion during native-only empty-body range work; deterministic Err/checkpoint context avoids timing assertions |
| Public YieldBlock | `yield-baseline`: `rootchildroot`; a recovered failure leaves `child` installed | `yield-fix`: renders `rootchild` exactly once and restores root context on panic, separately tested with race instrumentation |
| Numeric scan | `numeric-scan-baseline-multiply`: multiplying by a 1,000-byte zero string succeeds with 100 steps | Scan work is reserved before ParseInt/ParseUint/ParseFloat; fixed signed/unsigned/float cases terminate with ErrStepLimit |

The recursion probe uses small fixed source and a reduced Go stack; an external gate deadline supervises the process and bounds TERM/KILL cleanup. It is not an OS memory-isolation experiment. No OOM probe or dedicated fuzzing was run. An initial probe invocation used an invalid nested-module package path and was discarded; the correctly rooted baseline above supplies the evidence. An initial conversion fixture used equality, whose existing type rejection bypasses numeric conversion; the multiplication baseline/fix is the causal pair.

## Runtime coverage

| Matrix | Observation and tests |
| --- | --- |
| R01–R02 | Bounded child recursion tests, shared Runtime across composition and full inheritance traversal; content composition compatibility retained. Public reentry into ExecuteContext creates a new top-level render, outside restricted ownership. |
| R03 | Empty-body native range advances, expression trees, collection equality/cycles, pointer/interface traversal, aggregate truth checks, string equality and numeric string scans consume fuel/depth. |
| R04 | Pre-cancelled and expired contexts include empty templates; cancellation after a callback and custom causes on Go 1.20+ preserve errors.Is. |
| R05–R06 | Entry/release/exit channels demonstrate synchronous callback ownership; native-only work uses deterministic checkpoint cancellation. No interpreter worker/select is introduced. |
| R07 | Nested try and isset cannot catch cancellation/step/depth termination; recovered callback panics cannot reset the latch. Policy/value/output errors require their owning lanes. |
| R08 | Ordinary catch/rollback behavior and restored dynamic context pass, including failed nested try. |
| R09 | Text, escaped, raw, safe HTML, JSON and try commits preserve errors/short writes. Checked escaper adapters preserve nested callback reentry. |
| R10–R11 | Success/error/panic/cancel cleanup, pooled ranger resets, unrelated later execution and concurrent per-render budgets pass under normal and separate race instrumentation. |
| R12 | Nil context/negative depth fail before borrowing; zero is unlimited; one/exact boundaries and MaxUint64/large depth avoid arithmetic wrap. No safe defaults are implied. |

## Validation results

No GitHub Actions workflow exists on this fork branch. Local full validation was used rather than claiming CI results. Installed host-gate admitted every test/check/profile stage; queue wait was separate from command execution.

- `final-compatibility-normal`: `go test -count=1 -p=1 -parallel=2 -timeout=120s ./...` — pass.
- `final-compatibility-race`: same command with `-race` — pass, separately recorded.
- `final-compatibility-go116`: Go 1.16.15 with `-ldflags=-linkmode=external` and the same test flags — pass. The initial internal-link run passed the root package but modern macOS rejected two loader binaries for missing LC_UUID; the targeted external-link repair and final full external-link run pass, with linker warnings retained in logs.
- `correctness-submission`: standalone generic submission tree, `go test -race -count=1 -p=1 -parallel=2 -timeout=120s .` — pass.
- `recovery-coverage`: additional focused `-race` regressions require the exact step/depth sentinel through nested try, depth termination inside isset, and fuel exhaustion in content execution — pass. This adds a test file after the full stage; production inputs remain `d2f089c`.
- `vet`: `go vet ./...` — pass. No repository CLAUDE.md, .claude/rules or lint configuration exists; canonical available global guidance and upstream conventions were followed.

Full stages: 240-second per-stage execution deadlines, five-second subprocess TERM cleanup, aggregate 900-second gate deadline/ten-second cleanup. Focused tests used 120–180-second external deadlines and five-second cleanup. Profile/benchmark scripts used bounded per-stage subprocesses and exclusive admission. All failed setup/baseline stages remain explicit in their receipts.

## Dispatch performance and profiles

Diagnostic cached-execution fixtures in `execution_benchmark_test.go`, Go 1.27.3/darwin/arm64, GOMAXPROCS=2, one benchmark goroutine, exclusive host admission. Three independent 300 ms samples per mode; profiles use separate 500 ms samples. Positive limits are MaxSteps=1,000,000 and MaxDepth=128 with context.Background. These diagnose interpreter dispatch; they do not substitute for a live deadline/cancellable-context, p99/concurrency, compile or Karsto end-to-end program.

Median ns/op from retained upstream samples versus final `d2f089c`:

| Fixture | Upstream Execute | Fork Execute | Fork ExecuteContext | Final allocation counts |
| --- | ---: | ---: | ---: | --- |
| Small header | 76.01 | 115.3 | 123.4 | 24 B, 1 alloc in all modes |
| 8 KiB escaped body | 6370 | 6286 | 6272 | 24 B, 1 alloc in all modes |
| Silent 100-advance range | 1351 | 1145 | 1345 | 64 B, 3 allocs in all modes |
| Block composition | 2698 | 3129 | 3130 | 4728 B, 36 allocs (one sample 4729 B) |

The small header regresses about 62% (47 ns) versus upstream; this includes sink correctness and ownership checks as well as metering. It is not hidden by the roughly 8% median ratio across these four diagnostic fixtures. The full representative <=20% target remains lane 4's validation gate, not a result established here. The shorter silent loop path changed dispatch structure; the results are not a general throughput claim.

CPU profiles were inspected flat and cumulative, plus alloc_space, inuse_space, mutex and block profiles, before changes or attribution. Initial profiles showed escaping and top-level checked adapters allocating on each use, alongside repeated checkpoint/empty-list dispatch. Runtime-owned adapters with unwind-safe restoration remove those added allocation sites; empty bodies now avoid list dispatch while every advance remains checked/charged. Final alloc_space profiles are dominated by the existing block/scope allocation paths (executeYieldBlock about 68%, recover about 16%, newScope about 14%); adapters no longer appear as allocating sites. Final CPU samples place checkpoint at about 7.4% flat/11.6% cumulative and HTML escaping at about 16%. Remaining checkpoint cost is explicit interpreter enforcement. Mutex samples are tiny runtime events; block samples are benchmark/profile harness channel waits. Sampled inuse_space is zero, which is not a proof of zero retained heap or a process-memory bound. Peak RSS and application latency were not measured in this dispatch exercise.

## Remaining limits

Values/output allocation and builtin bounds, compile/load/input admission, capability policy and downstream side-effect prevention are not implemented by this lane. Unrestricted native callbacks, implicit formatting hooks, channels, custom rangers, writers and loaders can block or allocate; ExecuteContext waits for them. Very large/unlimited depth can exhaust the Go stack. Existing independent equality semantics (including array comparison behavior) were preserved, not folded into metering. Do not merge a tenant-safe claim or close Karsto #209 from these results.
