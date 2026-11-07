# Lane 2 execution evidence

## Current handoff

Execution is active on `work/lane-2-values`. Generic candidate code is `906e93856aea9dd70c98526a9bad34e39c5b6b85`, based on Lane 1's final explicit handoff `32997fd20e6986220c8c0b60bfff9cedc49d46b7` ([PR #3](https://github.com/ashtonian/jet/pull/3)). The earlier `86fce7a` meter was consumed first; the final base also includes the YieldBlock correction, numeric scan accounting and reusable checked sinks. Local backup refs preserve both pre-rebase task histories. No other lane checkout was edited.

The candidate adds enforced `MaxValueBytes`/`MaxOutputBytes`, `ErrValueLimit`/`ErrOutputLimit`, shared pre-allocation reservations, generated-output transactions, metered default bindings and bounded data-only formatting/JSON. `ConsumeSteps` and `ReserveValueBytes` are cooperative Runtime hooks for owned helper adapters; neither grants restricted permission. Values remain cumulative logical payload, not heap/RSS. Public options remain zero-unlimited and reject negative limits. The [cost inventory](lane-2-cost-inventory.md) records the schedule and unsupported capabilities.

`foundation-values-normal` passes the rebased candidate's focused normal workflow/compatibility checks. Earlier focused normal/race stages passed the implementation before the final base and diagnostic refinements. Final CI, Darwin Go 1.16, controlled reservation-on/off profiles and restricted-policy integration are pending; they are not inferred from those earlier passes. The unchanged CI workflow from Lane 4 `68bddd3fbd3fce1fce4f710400c6bd664db34d19` is consumed separately at `e8b80f1`. It tests Go 1.16.15 plus stable normal/race on Linux. Action pins match the official latest releases (checkout v7.0.1, setup-go v7.0.0).

### Lane 3 contract

- `builtinForValue(reflect.Value)` recognizes only private immutable descriptor pointers installed in the initialization registry; matching a type/name/program counter is insufficient. Locals, map/slice aliases and pipeline calls retain the descriptor. `meteredVariables` is selected for ExecuteContext; legacy Execute retains original function values. Public Runtime Resolve/MustResolve and direct native arguments preserve original functions for unrestricted callbacks. Function-valued containers are outside the data-only restricted profile.
- `builtinDescriptor.contract` classifies string, interpreter, output, JSON and diagnostic bindings. `builtinDiagnostic` (`dump`) must be denied by restricted policy before invocation. Other bindings have owned work/value implementations in `builtin_strings.go`, `builtin_json.go`, `execution_print.go` and `execution_values.go`.
- Audited generated values include private `jsonOutput`, `hiddenBool` and `*intsRanger`. Arbitrary Renderer/Ranger/SafeWriter, Formatter/Stringer/Error/Marshaler hooks and unsupported reflection remain native capabilities. Native formatting/JSON fallbacks call the canonical `beforeCall` before invocation; restricted policy must deny them. No policy selection is implemented in this lane.
- Runtime compilation limits and namespace denials returned by the loading bridge must latch termination before `includeIfExists` can treat an error as an absent file. The shared runtime's `terminate`/`reserve` path is used throughout; there is no duplicate meter/evaluator.

### Lane 4 contract

Karsto helpers at `50f35351fabea91c8bb4f3cbbb80b542171a6d19` are inventoried in the [helper handoff](lane-2-cost-inventory.md). Registered globals shadow defaults, so fixing defaults does not meter them. Three-argument replace, forgiving len, index byte-to-rune conversion, dict key formatting, eager variadics and number precision need explicit adapters. Owned adapters can call `a.Runtime().ConsumeSteps(n)` and `ReserveValueBytes(bound)` before native work/construction. Checked arithmetic is the adapter's obligation; an after-return reservation cannot prevent native allocation. Registration/admission by audited immutable identity remains Lane 3's policy API, not automatic RegisterFunc approval.

Karsto integration, packaging and actual webhook/chain/PDF acceptance remain Lane 4's work. The candidate is not tenant-ready from lane 2 tests. All work stays in the fork; the owner has directed that no new upstream PRs be opened. Internal assignment documents remain excluded from eventual upstream submission diffs.

Upstream [#234](https://github.com/CloudyKit/jet/pull/234) remains separate at `f300a9af0cdcf75d9ea90516d8f915b45bd127f9`; its dump globals/nil-context fix was inspected, not recreated here.

## Baseline fixtures and observations

Tests use the real `Set.Parse` / `Template.Execute` path and pinned fastprinter `33d98a066a53`. `value_output_test.go` covers every default binding in text and HTML mode, plus concatenation, shared children, escaping and nested try/rollback. `value_output_probe_test.go` supplies fixed fixtures to reuse for the metered fix. Every native string input/result is small and finite: the largest result is 128 KiB, generated via 16 doublings or a fixed repeat/replace. No OOM/stack-overflow or giant-precision input was executed.

The probe writer refuses any write exceeding its **1,024-byte** sink allowance, without panicking. It exposes why a final sink limit is insufficient:

| Fixture | Result / final sink | Normal render allocations observed | Missing enforcement demonstrated |
| --- | --- | --- | --- |
| `doubling` | `131072`, six bytes, success | 558,248 bytes / 94 allocations | Native intermediate string growth precedes output. |
| `repeat` | `131072`, six bytes, success | 133,272 / 34 | Native builtin allocates before final sink. |
| `replace` | `131072`, six bytes, success | 141,520 / 36 | Replacement expansion precedes final sink. |
| `split` | `4096`, four bytes, success | 71,576 / 34 | New string slots allocate despite tiny output. |
| `nestedTry` | 16,384 attempted, zero accepted, **success despite refused commit** | 66,104 / 16 | Nested retention before sink; independent swallowed commit-error defect addressed by Lane 1's correctness patch. |
| `rollback` | `ok`, two bytes, success | 182,720 / 72 | Four generated 16 KiB strings/try buffers disappear before final sink. |
| `discard` | `ok`, two bytes, success | 19,592 / 47 | `exec` generation bypasses final sink. |

Allocation deltas include runtime/formatting overhead and are observations, not logical-budget costs or stable test assertions. Passing these compatibility/baseline tests does not pass V01–V08; value/output options do not exist at the tested baseline. The metered comparison must reuse the exact fixtures, set configuration, dependencies and toolchain, apply fixed positive logical budgets and reject before the offending operation. The nested commit-error result is distinct from allocation prevention.

## Allocation profile interpretation

The separate normal-build allocation-profile run uses `-memprofilerate=1`. Both flat/cumulative top tables and source-annotated `evalAdditiveExpression`, `executeTry` and `fastprinter.PrintValue` were inspected.

- `evalAdditiveExpression` accounts for 543.86 KiB cumulative / 256.62 KiB flat across these fixtures. Source attribution is at `left.String() + fmt.Sprint(right)`; `fmt.Sprint` adds 287.23 KiB cumulative / 128 KiB flat. Reserving only the concatenated string while retaining an unbounded formatter would miss a separate destination.
- `bytes.growSlice` accounts for 156 KiB flat; `executeTry` accounts for 242.89 KiB cumulative including child generation. Reserving at the final sink is later than temporary buffer growth.
- `strings.explode` allocates 64 KiB of slots in the split fixture; child string bytes need no recursive re-charge.

No CPU/performance-overhead conclusion is drawn from these allocation observations. The profile is not an RSS cap or evidence of pre-allocation rejection; that evidence remains a required fixed-runtime comparison. Lane 4 owns the final performance report.

## Commands and durable receipts

All paths below are relative to this checkout's `.git/lane-2-validation/`. Each stage has `inputs.json`, `run.log` and `result.json`, with HEAD, SHA-256 source/test/fixture manifest (including untracked tests), toolchain/configuration, exact command, deadline/cleanup and exit result. Input manifests were checked unchanged after execution. Queue waiting is separate from execution. Installed host-gate supervision provides its own managed command receipt and process-group cleanup.

Toolchain: `go1.27.3 darwin/arm64`; module still declares Go 1.16; no dependencies or language-version changes. Default focused pool grants `GOMAXPROCS=2`, `-p=1`, `-parallel=2`. `GATE_LOCK_COMMAND_TIMEOUT=120` and `GATE_LOCK_COMMAND_KILL_AFTER=5` bound baseline stages; tests additionally use `-timeout=60s`.

| Stage | Command after `gate-lock run-focused <label> --` | Result |
| --- | --- | --- |
| `compatibility-baseline` | `go test -p=1 -parallel=2 -timeout=60s -count=1 -run '^TestValueOutputCompatibility$' .` | Pass. |
| `normal-baseline` | `go test -p=1 -parallel=2 -timeout=60s -count=1 -v -run '^TestValueOutput(Compatibility\|Amplification)$' .` | Pass; table above from this run. |
| `amplification-baseline` | `go test -p=1 -parallel=2 -timeout=60s -count=1 -v -run '^TestValueOutputAmplification$' -memprofile=<artifact>/allocs.pprof -memprofilerate=1 -outputdir=<artifact> .` | Pass; allocation profile collected. Generated `jet.test` moved into this stage's artifact directory. |
| `race-baseline` | `go test -race -p=1 -parallel=2 -timeout=60s -count=1 -run '^TestValueOutput(Compatibility\|Amplification)$' .` | Pass, separate from normal/profile results; 180-second execution deadline. |

Profile inspection uses `gate-lock run-check` with a 60-second execution / five-second cleanup bound: `go tool pprof -top -alloc_space`, `-top -cum -alloc_space`, and `-alloc_space -list 'evalAdditiveExpression\|executeTry\|PrintValue'` against the preserved binary/profile. Stages `allocation-profile`, `allocation-profile-cumulative` and `allocation-profile-lines` all passed.

Artifact hashes:

| Artifact | SHA-256 |
| --- | --- |
| `normal-baseline/run.log` | `0a72bc995e5f6aa5b8ea42e4c789d570b56f5fc12a5b7398cce0c4208a02a9e8` |
| `normal-baseline/result.json` | `9d501acedbc5beac58cd655d6f0aca063ef4569f2d0dea289a27018cc3a4b3a2` |
| `amplification-baseline/allocs.pprof` | `7890de839ca0cd28c58af66ad6d1780bfdde041f15087982bcc98e4cd2ee1eb3` |
| `amplification-baseline/result.json` | `a8972575f7761848d436b4fd44f5090a3d320a590b0dfa403701434d1d687954` |
| `race-baseline/result.json` | `4a691256ac535727db3c1b6c605fe7fd42839c3cb96ee5b40551ff5007c332fb` |

Full upstream suites/CI, oldest-supported-Go validation, V01–V08 fixed-runtime acceptance, R07/R09 dependent paths and actual Karsto workflow validation remain unexecuted. No upstream submission, packaging/release, tenant-safe claim or issue closure is established by a baseline pass. Internal lane documents stay out of eventual upstream submission diffs.
