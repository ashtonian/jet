# Acceptance and performance matrix

The rows below define required acceptance; they are not a pass ledger. Current evidence, reproduced failures and unexecuted final coverage are recorded in the [2026-11-08 final review](final-review.md). Owners must record the exact tested SHA, dirty/untracked input manifest, dependency/toolchain configuration, command, execution deadline, log/receipt and result. Missing or skipped coverage is not a pass. Tests should exercise actual supported templates, not implementation-shaped counter assertions.

## Runtime (Lane 1)

| ID | Input / workflow | Required observation |
| --- | --- | --- |
| R01 | Self yield `{{ block b() }}{{ yield b() }}{{ end }}` | Baseline crashes in bounded child; same source with finite depth returns errors.Is(ErrDepthLimit) without process failure |
| R02 | Mutual yield, content recursion, Runtime.YieldBlock, include, exec, includeIfExists | All native nested paths consume one allowance; no reset by switching templates or discarding output |
| R03 | Non-writing range with empty body; long expression/collection equality | Finite fuel terminates even without a writer/callback; recursion and data traversal are covered |
| R04 | Pre-cancelled, mid-execution and expired-deadline contexts, empty template | Native work stops; inspect context error and supported custom cause; no wall-clock timing dependency in unit tests |
| R05 | Controlled callback signals entry, waits for release, then exits | Cancellation does not make ExecuteContext return before callback exits; explicit release/drain proves ownership without leaking work |
| R06 | Native cancellation during non-writing work with synchronization | No interpreter work after return; a tick callback alone is insufficient evidence for silent native loops |
| R07 | Nested try/catch and isset around cancellation/each limit/policy error | Terminal error reaches caller; catch does not execute or convert it to false/success |
| R08 | Ordinary runtime error inside try/catch | Existing catch behavior/output rollback retained; scopes/context/content restored |
| R09 | Writer failure and short write through text, escaped/raw/safe/JSON output and try commit | errors.Is reaches original cause or io.ErrShortWrite; execution cannot later report success |
| R10 | Repeated success, error, panic and cancellation followed by unrelated render | Pool state/caller references cleared; no scope/function/context/output contamination |
| R11 | Concurrent renders using one parsed template and shared cache | Deterministic per-render scopes, budgets and functions; no races in supported usage |
| R12 | Limits at zero/negative/one, exact boundary, large integer values | Defined validation, no counter arithmetic wrap or unintended zero-as-safe interpretation |

Use deterministic context/callback synchronization or an injected checkpoint seam for cancellation. Real deadlines are integration/performance measurements. External timeout is a harness guard, never the assertion that the interpreter is cancellable.

## Values and output (Lane 2)

| ID | Input | Required observation |
| --- | --- | --- |
| V01 | `s := "ab"; s = s + s` inside a non-writing loop | Reservation fails before the offending allocation; source/budget held constant for baseline and fix |
| V02 | repeat/replace/split/case/escape/JSON, huge width/precision | Proven pre-allocation bounds, terminal rejection; bounded invalid argument behavior |
| V03 | Nested map/slice/array creation, aliasing and shared child values | Container slots accounted; no exponential size traversal; cycles denied or bounded |
| V04 | Output loop, including HTML expansion | Limit applies to generated escaped bytes; fuel and output independently demonstrated |
| V05 | Nested try buffers, failed try, repeated rollback | Buffers bounded before growth; terminal error cannot be caught; no output double charge at commit |
| V06 | exec with output discarded | Generated output still charged; discard does not reset compute/value/output allowances |
| V07 | Near-overflow sizes, ranges, zero/negative repeat counts | Checked arithmetic, no catastrophic allocation; errors defined |
| V08 | All supported helpers and escaping modes under sufficient budget | Same result as baseline; explicit policy denial for unsupported hooks |

Allocation profiles/RSS plus a pre-reservation observation establish amplification control; checking an already produced value's length does not. Use small decisive allocation bounds, not an attempt to OOM the host.

## Compilation and capabilities (Lane 3)

| ID | Input | Required observation |
| --- | --- | --- |
| P01 | Self/mutual/mixed import and extends cycles, alias paths | Adopt #236 coverage, bounded child baseline; valid diamonds remain valid |
| P02 | Deep finite import chain, nested if/range, parenthesized/unary/expression chains | Limit before parser stack exhaustion; execution depth alone is insufficient |
| P03 | Source bytes N-1/N/N+1, huge literal, many small AST nodes | Inline/named/runtime source and AST admission enforce their distinct budgets |
| P04 | Many dependencies with each under the per-file cap | Total graph bytes/template count capped; dynamic includes cannot reset allowance |
| P05 | Cancellation mid-scan, parse error, token-send failure, bounded reader error | Lexer/reader completed before return; no worker or retained source |
| P06 | Loose-policy cache warmup then stricter request, concurrent engine policies | Cache hits do not bypass admission; no cross-engine or cross-tenant contamination |
| P07 | Relative names, extensions, custom delimiters/comments, Unicode, development mode | Syntax/loader compatibility and correct error identity |
| C01 | Funcs through globals, nested vars, aliases, methods and returns | Restricted policy denies before invocation; trusted mode retains supported behavior |
| C02 | Custom Renderer/Ranger/SafeWriter/Stringer/Marshaler and reflective conversion | Every implicit invocation site is covered; denied hooks are never called |
| C03 | Cyclic pointer/interface/map/slice inputs and giant data | Admission terminates within bytes/items/depth/work budget; no unbounded reflect walk |
| C04 | Channel range and callback reentry into Execute | Restricted mode denies unmeterable ownership paths |
| C05 | Concurrent global updates, builtin shadowing and capability aliases | Immutable authorized bindings; no name-based privilege escalation |
| C06 | Unauthorized include names and foreign cached templates | Authorize canonical namespace before cache; tenant data/templates stay isolated |
| C07 | Custom blocking writer/loader in unrestricted mode | Ownership boundary documented and controlled test cleaned up; no false interruption guarantee |

## Downstream workflows (Lane 4)

| ID | Workflow | Required observation |
| --- | --- | --- |
| I01 | Render/RenderString, named/inline, text/HTML, cold/hot cache | Preserved Renderer contract and semantics; empty result on error, no worker after return |
| I02 | Real wrapper under non-writing native loop, recursion and amplification | Matching baseline/fix reproductions through public Engine APIs; low-level Jet tests alone insufficient |
| I03 | Webhook body and any header fails rendering | No delivery after failure; partial headers/body not sent; error classified visibly |
| I04 | Every chain node category consuming ResolveTemplate; nil Renderer on dynamic input | Error propagated before mutation/dispatch; neither raw source nor empty substitution; static literal path explicit |
| I05 | PDF HTML template fails | No PDF subprocess launched; template safety does not certify external renderer isolation |
| I06 | Config/default/helper migration, invalid budgets, cause mapping | Inspectable errors.Is plus current Karsto E-code contracts; supported helper results unchanged |
| I07 | Concurrent engines, tenant namespaces, cache eviction/disable/close | No environment/function/policy leakage; compile cache retains only immutable allowed state |
| I08 | External consumer of released fork module | No local replace needed; no hidden upstream/fork type mixture; immutable dependency resolves |

## Bounded baseline protocol

Run crash and large-allocation probes in child processes with externally enforced execution and memory/stack bounds and bounded TERM/KILL cleanup. `debug.SetMaxStack` may reduce a child stack-overflow probe's footprint; use the same setting for baseline/fix. A Go heap target is not a hard memory cap. Use OS process/container resource isolation when the risk needs it; container startup itself requires host admission. Capture small durable stderr excerpts and exit status. Never intentionally leave a runaway loop after its parent test returns.

For the legacy wrapper ownership repro, use a finite blocking callback with entry/release/exit channels, cancel only after entry, observe return ordering, then release and join. Separately reproduce native silent computation. Use the original source/configuration/dependencies; avoid interpreting a race-only behavior as the normal build's cause. Unit tests avoid sleeps; latency experiments may use real time and report noise.

## Performance program

Compare four modes on identical fixtures: pinned upstream; fork `Execute` with no budgets; fork `ExecuteContext` with realistic positive budgets; Karsto's canonical renderer. Report compile-only, cached execute, uncached parse+execute and end-to-end separately.

| Fixture | Required variants |
| --- | --- |
| Small webhook header | Literal and interpolation, text and HTML |
| Medium webhook body | Fixed 8 KiB and 64 KiB result targets, bounded data collections |
| Nested loops | Fixed shapes and operation counts, writing and non-writing |
| HTML escaping | Mostly-safe text and expansion-heavy characters |
| String operations | Concatenation, replace/join/case conversion, bounded amplification rejection |
| Template composition | Include, inheritance, block/content and try transactions |
| Concurrent rendering | Shared compiled template and multiple engines; fixed concurrency 1, 2, 8 and host-fit maximum |

Capture ns/op, B/op, allocs/op, p50/p99 latency, throughput, sustained CPU, peak RSS, and cancellation overshoot (time and extra checkpoints after cancellation; time for native-only versus cooperative callback separately). Use fixed input corpus/seed, warmup, fixed-duration samples and multiple independent runs. A load generator capped below capacity must not be mistaken for equal performance. Record fixture sizes, metering options, cache state, GOMAXPROCS, Go version, GOOS/GOARCH and instrumentation.

Collect CPU, alloc/heap, mutex and block profiles on representative hot paths; inspect flat and cumulative costs before attributing overhead. Profile runs and race runs are distinct from benchmark samples. Use benchstat/confidence intervals where suitable; report per-workload ratios and a median of representative execution ratios without hiding p99 regressions. Target <=20% median execution overhead; do not apply that number to uncached compile+execute as if it were the same metric.

Correctness takes priority over the target. If missed, identify the structural cost in profiles and retain safety while improving it. No result or budget default is established by this planning document.
