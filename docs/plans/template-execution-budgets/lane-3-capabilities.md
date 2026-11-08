# Lane 3 capability inventory and binding handoff

Execution is authorized; this inventory is preparation for enforcement. **No restricted mode is enabled by this document or the compilation API.** Runtime baseline is `202464510d5b57f0afa9c5563f75babc1616682c`; lane 1's published runtime branch has been inspected through `d2f089ce4f617dd512c7eee6b5d825f2c0b57914`. Lane 2's [cost inventory](https://github.com/ashtonian/jet/blob/work/lane-2-values/docs/plans/template-execution-budgets/lane-2-cost-inventory.md) owns builtin implementations and their pre-allocation work schedule.

## Invocation dispositions

| Entry | Existing invocation | Required restricted enforcement and regression |
| --- | --- | --- |
| `exec.go:Template.Execute`, `ExecuteContext` | Borrowed vars/data, runtime Set globals | Snapshot vars/data under positive input byte/item/work/depth budgets; no callable values, pointers, structs, channels or method-bearing types. Globals are immutable admitted data/bindings in a restricted Set. Caller must synchronize all input mutations during admission; copying cannot protect an unsafe concurrent Go map access. |
| `eval.go:Runtime.resolve` | Vars, parent vars, globals, default variables | Values resolve to data or opaque audited binding objects. Names grant no authority; shadowing a name cannot create a binding. Restricted defaults must come from lane 2's audited registry. |
| `eval.go:evalPipeCallExpression`, reflected `Call` | Jet Func or arbitrary Go function | Deny all caller functions before argument conversion/invocation. An internal opaque binding owns an audited implementation and immutable disposition. Function program counters cannot identify closures and must not be used. Test aliases, nested values and callable returns. |
| `eval.go:resolveIndex`, `evaluateArgs`, `indirect` | Method discovery, conversion and pointer/interface traversal | Data-only snapshots have no methods or pointers. Add a runtime denial before method discovery and unsupported conversion as defense in depth; do not invoke a method to decide whether it is safe. |
| `eval.go:executeList` | `Renderer.Render`, otherwise fastprinter | Deny external Renderer; allow only explicit internal renderer objects with lane 2's bounded serializer contract. Data-only print path cannot reach Stringer/Error/Formatter. |
| `eval.go:evalSafeWriter`, `escapeWriter` | SafeWriter function applied to formatted arguments | Opaque internal binding selects a vetted writer; external SafeWriter aliases denied. Set escaping must also be vetted; arbitrary WithSafeWriter callback is outside restricted policy. |
| `eval.go:evalAdditiveExpression` | fmt conversion, byte-slice conversion, concatenation | Data-only operands plus lane 2's metered formatter/concatenation; no Stringer/Formatter/Error callbacks. |
| `ranger.go:getRanger`, `Range`, `ProvidesIndex` | External Ranger, synchronous channel receive, pooled native ranges | Deny channels/external rangers before acquisition. Allow admitted native collections and a sealed internal ints ranger with lane 1 advancement checkpoints. |
| `default.go:json`, `writeJson` | Marshaler/TextMarshaler, JSON renderer | Data-only bounded serializer from lane 2; reject native marshaling interfaces before invocation. Test nested hooks and values returned by builtins. |
| `default.go:dump` and fmt diagnostics | `%#v`, context/global/scope formatting | Explicit restricted denial. Error diagnostics must not format rejected values while reporting denial. Upstream #234 remains separate. |
| `eval.go:executeInclude`, `default.go:exec`, `includeIfExists` | Loader/cache and nested execution | Use one execution-owned parse admission and authorized immutable bundle. Check canonical namespace before cache. Missing optional include alone may become hidden false; cancellation, parse limits and policy denial remain terminal. |
| `set.go` and `loader.go:Loader` | Exists/Open/Read/Close, optional context-aware calls | Generic APIs own calls synchronously and cannot force arbitrary blocking code to return. Restricted sources must be sealed bounded in-memory bundles, not a claimed cooperative external loader or abandoned worker. |
| `Runtime.YieldBlock`, host callbacks and public reentry | Calls interpreter again; public Execute can start another execution | Host callbacks denied in restricted mode. Template-native composition uses lane 1's single execution state and lane 3's single compilation admission. No callback can obtain a new allowance. |
| `io.Writer`, custom escaping | Synchronous external callback | Library cannot preempt an arbitrary writer. Restricted integration must use its owned bounded memory sink; a template profile alone is not an OS isolation boundary. |

## Input snapshot contract

Initial implementation admits primitive scalar values, strings, arrays/slices and string-keyed maps into method-free execution-owned collections. Arrays/slices and maps are copied under admission; immutable strings may be shared. Noncyclic shared children are admitted once per copied occurrence, so a small DAG cannot amplify work without consuming admission. Active map/slice identities distinguish cycles from valid sharing. Pointers, structs, functions, method-bearing values, channels and unsafe/uintptr values are denied before invocation. Explicit trusted adapters must produce admitted data; this is not an automatic reflection adapter for time.Time or arbitrary application structs.

Every allocation reserves collection slots/key bytes first. The logical byte allowance is not a heap/RSS limit. Nil and empty collections retain defined rendering semantics. The snapshot API stays private until it is connected to an enforced restricted execution entry point.

## Required lane 2 binding contract

Authorization must use an unexported, opaque binding object owned by Jet, never a Go function's address or a string allowlist. The binding carries its callable/writer/renderer/ranger role and the concrete audited implementation. Aliases of that same object remain authorized; ordinary funcs under the original name remain denied. No public constructor accepts arbitrary callbacks as audited. Each produced value is either admitted data or a sealed internal object with an explicit runtime disposition. Unknown roles and new AST/value kinds fail closed.

Lane 2 owns the supported default registry and metered helper adapters; lane 3 owns immutable selection and policy checks. Required adapter surface for lane 4 is a finite set of audited helper descriptors/configuration, not `RegisterFunc` approval. The precise binding Go type and helper constructors must land against the same runtime SHA before J08 can enable the profile. No placeholder registry is treated as completed enforcement.

## Dependency gates

- J07 compile/load limits can be reviewed independently. Runtime includes/exec need lane 1's loadTemplate bridge and shared allowance field in a dependent commit.
- J08 remains blocked on J05's actual audited bindings/value coverage, not merely its inventory or size helpers. Continue input admission and fixtures while waiting.
- C01–C07 require executed tests at every row above and restricted/unrestricted concurrent use before tenant adoption. No acceptance row is checked by this inventory.

## Reviewed registry candidate and current blocker

Read-only review of `e883796c80058ad4f0c92458d6b2e6e64e9e7ec9` confirms `builtinForValue(reflect.Value)` accepts only private descriptor pointers in the initialization registry. `builtinDescriptor.contract` distinguishes string/interpreter/output/JSON/diagnostic roles; dump is diagnostic and must be denied. `meteredVariables` is selected for context-aware execution; public Runtime/Arguments conversions still preserve original functions for unrestricted callers. J08 must close any such conversion route when selecting the profile. Internal generated values are `jsonOutput`, `hiddenBool` and `*intsRanger`; arbitrary analogous interfaces remain denied. Native JSON/format fallbacks already call beforeCall and need the policy denial there.

The candidate is not accepted: [Go 1.16 JSON sizing fails](https://github.com/ashtonian/jet/actions/runs/42426825673/job/128566779953), reporting 27 versus 35 bytes. Corrected bounds, normal/race/minimum-Go results and cost evidence are prerequisites for enabling the profile. Source symbols above refer to [execution](../../../eval.go), [arguments](../../../func.go), [ranges](../../../ranger.go), [defaults](../../../default.go), [loading](../../../set.go) and [input preparation](../../../input.go); the candidate's additional builtin files remain in its recorded peer SHA.

All work/review stays in ashtonian/jet under the owner's later instruction. No new CloudyKit PR is authorized.

The JSON bound is corrected in `5583ec15361aea8f98c7a0d5a1fcbef25d407c0e`, whose Go 1.16/stable normal/race CI passes. Dependent capability implementation may consume that exact candidate; cost/profile and tenant acceptance remain distinct gates. Reuse the existing fork reviews.
