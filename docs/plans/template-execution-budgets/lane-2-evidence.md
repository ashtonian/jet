# Lane 2 execution evidence

## Current handoff

The production candidate is `5583ec15361aea8f98c7a0d5a1fcbef25d407c0e`, based on Lane 1's explicit final foundation `32997fd20e6986220c8c0b60bfff9cedc49d46b7` ([fork PR #3](https://github.com/ashtonian/jet/pull/3)). [Fork draft PR #2](https://github.com/ashtonian/jet/pull/2) carries the consolidated lane, final test/documentation commit and its exact CI result. No production code changed after `5583ec1`; the final batch adds byte-option boundaries and independent fuel/output loop checks and discarded-output termination through isset. Earlier foundation/rebase histories have local backup refs. No other lane checkout was edited.

Implemented: checked `MaxValueBytes`/`MaxOutputBytes` reservations, inspectable `ErrValueLimit`/`ErrOutputLimit`, shared sticky termination, generated-output transactions, private immutable builtin bindings, bounded data-only formatting/JSON, allocating conversion checks and bounded diagnostics/numeric parse errors. Public [execution costs](../../execution-costs.md) and the [cost inventory](lane-2-cost-inventory.md) define the logical schedule. No heap/RSS, unrestricted-host-code or tenant-safe execution claim follows from these limits.

All work stays in the fork under the owner's publication hold. Do not open CloudyKit PRs/issues/submissions. Internal `docs/plans/`, RFCs and fork CI stay out of any future generic submission. Existing fork reviews are reused; no issue closure or release is authorized by this evidence.

## Acceptance status

| Gate | Executed evidence / remaining limit |
| --- | --- |
| V01 | Same finite doubling fixture and positive options with only value reservation disabled/enabled; source profiles show rejection before the next Builder.Grow. |
| V02 | Repeat/replace/split controls; case/escape/JSON and encoding-size regressions. Checked size helpers cover near-overflow without constructing giant values. Arbitrary width/precision formatting is native capability; Karsto adapters remain Lane 4. |
| V03 | New map/slice slots, shared children, map key conversion once, data-only printing/JSON cycles bounded by depth. Input admission/cycle rejection remains Lane 3. |
| V04 | `TestExecutionOutputLoop` holds an escaped 100-iteration loop constant and independently exhausts output or steps; escaped boundary cases cover partial prefixes. |
| V05 | Nested copies, exact no-double-output-charge commit, repeated rollback, terminal catch/isset paths; paired output control profiles show buffer growth prevention. |
| V06 | Discarded exec fixture succeeds only with output reservation disabled; enabled rejection precedes caller-sink writes. Shared execution state retained. |
| V07 | Checked add/multiply/encoding sizes; repeat zero/negative; int64 endpoint/range distance; invalid reflection conversion returns ordinary catchable errors. |
| V08 | All 25 default bindings plus text/HTML/escaping/alias/native compatibility pass with sufficient budgets. **Restricted denial of unsupported hooks is still Lane 3's dependent gate.** |
| R07/R09/R12 | Sticky value/output failures through catch/isset; partial/error/short writes across text/raw/safe/JSON/try and recovering callbacks; zero/negative/one/exact/MaxInt64 byte options. Policy errors await Lane 3. |
| Compatibility | Linux Go 1.16/stable normal/race full suites pass at `5583ec1`; Darwin Go 1.16 full suite passes, with focused checks for final new tests. Final branch CI is recorded in PR #2. |
| Adoption/performance | C01–C07 restricted closure, actual Karsto I01–I08 workflows, packaging and the 20% performance report remain external gates. No admission/profile is enabled by lane 2. |

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

Allocation deltas include runtime/formatting overhead and are observations, not logical-budget costs or stable test assertions. These baseline observations alone do not pass V01–V08: value/output options did not exist at that source. The paired reservation controls below isolate the new guard from other runtime/formatting changes. The nested commit-error result is distinct from allocation prevention.

## Controlled reservation comparisons

[Paired proof](lane-2-evidence/paired-proof.json) verifies matching source/test/dependency/configuration/toolchain inputs except **one reservation line** per pair. Both runs use `ExecuteContext`, value allowance **32,768**, generated output **1,024**, steps **1,000,000**, depth **100**, the same 1,024-byte refusing sink and fixed fixture sources. All paired source manifests remained stable; controlled mutations were restored. Expected control test exit 1 is an observed failed protection assertion, not an infrastructure success claim.

The value control changes only the value reserve's configured-limit argument to zero; the output control changes only the corresponding output reserve argument. Other limits remain enabled. Patches are committed with [value](lane-2-evidence/value-control-final/control.patch) and [output](lane-2-evidence/output-control-final/control.patch) receipts. This isolates reservation causality from the new formatting implementation and from race instrumentation.

| Fixture | Reservation disabled: outcome, allocated bytes / count | Enabled: outcome, allocated bytes / count |
| --- | --- | --- |
| doubling | Success, six sink bytes; 263,704 / 49 | ErrValueLimit before sink; 17,768 / 39 |
| repeat | Success, six sink bytes; 131,760 / 13 | ErrValueLimit before sink; 224 / 8 |
| replace | Success, six sink bytes; 139,984 / 16 | ErrValueLimit before sink; 8,432 / 11 |
| split | Success, four sink bytes; 70,344 / 15 | ErrValueLimit before sink; 4,336 / 10 |
| nestedTry | Later ErrValueLimit; 50,488 / 25 | ErrOutputLimit before retention/sink; 21,832 / 23 |
| rollback | Later ErrValueLimit; 45,464 / 17 | ErrOutputLimit before retention/sink; 21,192 / 19 |
| discard | Success, `ok`; 23,208 / 49 | ErrOutputLimit before sink; 23,800 / 52 |

Normal-build `-memprofilerate=1` profiles were read as flat, cumulative and source views for every pair ([inspection commands/receipts](lane-2-evidence/profile-inspection-stages.json)). The per-render allocation observations above include runtime overhead, are not logical charges and are not stable test assertions.

- Value control: 524 KiB in `internal/bytealg.MakeNoZero`, plus 64 KiB `strings.explode` slots. Enabled: MakeNoZero is 28 KiB and explode is absent. `concatString` is 256 KiB cumulative in control versus 16.09 KiB enabled; the enabled source view attributes rejection to `valueLength` before the next `Grow`. The remaining repeat allocations are the small input prerequisites, not the rejected 128 KiB result. Split fails at `reserveSlots` before string-header allocation. See [control source](lane-2-evidence/value-control-final/source.txt) and [enabled source](lane-2-evidence/value-guarded-final/source.txt).
- Output control: 56 KiB `bytes.growSlice`; enabled has no `bytes.growSlice` allocation. Both still construct 48 KiB across the three repeat prerequisites. The enabled source view terminates at generated-output reservation before retained-buffer reserve/write. See [control flat](lane-2-evidence/output-control-final/flat.txt), [enabled flat](lane-2-evidence/output-guarded-final/flat.txt) and [source](lane-2-evidence/output-guarded-final/source.txt).
- Child peak RSS (value control/enabled): 12,091,392 / 11,878,400 bytes; output: 12,042,240 / 12,124,160. These mostly reflect the process/runtime. Test regex/harness allocations also differ, so profile totals/RSS do not establish execution overhead or a memory cap. No CPU/latency optimization claim is made. Lane 4 owns the performance program.

The original upstream profile separately attributes 543.86 KiB cumulative / 256.62 KiB flat to `evalAdditiveExpression`, 287.23 KiB cumulative / 128 KiB flat to `fmt.Sprint`, 156 KiB to `bytes.growSlice` and 64 KiB to split slots. Those explain the old allocation sites; only the paired new-runtime controls isolate guard causality.

## Independent root-cause regressions

- Diagnostic amplification: the same 16-level shared DAG produced 262,203–262,248-byte errors for assignment/map/ints; an invalid 16 KiB range value produced a 16,455-byte error. Fixed errors are 55–125 bytes and never invoke arbitrary formatting hooks. Ordinary metered diagnostics use positions/types rather than expanding scopes or values. [Failing baseline](lane-2-evidence/diagnostic-baseline/run.log), [passing fix](lane-2-evidence/diagnostic-fix/run.log).
- Argument conversion: lower(map), repeat(slice count) and split(slice) escaped as string panics; byte-slice map key evaluation ran twice before panic. Fixed conversion validation returns ordinary errors and evaluates/converts/inserts the key once. [Baseline](lane-2-evidence/argument-baseline/run.log), [fix](lane-2-evidence/argument-fix/run.log).
- Numeric parsing: a fixed 4,097-byte invalid numeric string allocated native error input copies without reservation. Now the parser reserves twice the input size before invocation; low allowance returns ErrValueLimit, sufficient allowance preserves errors.Is(strconv.ErrSyntax)/errors.As(*strconv.NumError) with bounded diagnostic text. [Baseline](lane-2-evidence/numeric-error-baseline/run.log), [fix](lane-2-evidence/sink-and-numeric-errors/run.log).

## Validation and reproduction

[Portable evidence index](lane-2-evidence/index.md) contains source manifests, commands, logs, deadlines/cleanup and receipts. Raw originals, profile binaries and managed host-gate receipts remain in this checkout's `.git/lane-2-validation/`; raw artifact hashes are committed. Published text replaces local paths/host identifiers and removes other owners' queue notices. No tokens or private host locations are published.

Every local stage uses installed host-gate with explicit execution timeout and five-second TERM escalation; queue waiting is separate. Named-package tests use focused admission, `GOMAXPROCS<=2`, `-p=1`, `-parallel=2`, plus Go test timeout. Profile/control stages use 120-second execution / 60-second test bounds. Full Darwin Go 1.16 uses all-host admission (300-second outer bound), 180-second test stage / 120-second Go test timeout, and bounded process-group TERM/KILL cleanup. There was no dedicated fuzzing or giant/OOM probe; the largest fixture result is 128 KiB.

| Stage / immutable input | Result |
| --- | --- |
| `normal-baseline`, `race-baseline`, `amplification-baseline` | Original compatibility/finite workflow observations pass; normal/profile/race results kept separate. |
| `value-control-final`, `output-control-final` | Expected test failures (exit 1); disabled reservation violates fixed protection assertions. Source restored. |
| `value-guarded-final`, `output-guarded-final` | Pass; same production candidate, options, fixtures and toolchain. |
| `final-core`, `final-vet` | Focused runtime/value/builtin/escaping checks and `go vet ./...` pass on Go 1.27.3 darwin/arm64. |
| `go116-final-platform` | Full Go 1.16.15 darwin/arm64 suite passes with external linker; original LC_DYSYMTAB warnings retained. Toolchain archive SHA-256 `28365c5c252970c10f2627dc0aa63a2ec3df04a92df4d45ed83dbe464732c3e0`; own SDK/compiler/link hashes recorded. SDK reused by APFS clone without writing the source checkout. |
| `final-terminal-coverage`, `go116-final-options`, `go116-final-terminal` | Final independent loop, byte-option and discard/isset tests pass on current Go and Go 1.16, respectively. Previous production/fixture stages remain valid; only added tests needed incremental Darwin execution. |
| Linux [run 42428373277](https://github.com/ashtonian/jet/actions/runs/42428373277), SHA `5583ec1` | Go 1.16.15 normal, stable normal and stable race full suites pass. Final added-test commit CI is recorded in PR #2. |

Failure history is retained: the earlier [Go 1.16 job](https://github.com/ashtonian/jet/actions/runs/42426825673/job/128566779953) exposed JSON backspace/formfeed escaping differences (27-byte bound versus 35-byte encoding). `5583ec1` corrects the cross-version bound; passing minimum-Go CI demonstrates the fix. An initial guarded profile run passed tests but failed to write its profile due a relative artifact path; final paired runs use absolute artifact paths. A first incremental Go 1.16 command was rejected before execution because run-focused requires a direct `go test` command; the corrected environment/direct command passed. A final documentation check also found over-deep source links in the inventory; those links were corrected before the passing final doc check. None of these failed stages are counted as passing coverage.

The compatibility workflow is consumed unchanged from Lane 4 `68bddd3fbd3fce1fce4f710400c6bd664db34d19` in `e8b80f1`. Its pinned actions matched official stable releases at adoption (checkout v7.0.1, setup-go v7.0.0). It has explicit job/command cleanup deadlines and does not duplicate same-repository PR and push jobs. Module Go version/dependencies remain unchanged.
