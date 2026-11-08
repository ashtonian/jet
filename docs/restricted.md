# Restricted rendering

`NewRestrictedSet` explicitly selects an immutable in-process capability profile. Existing `NewSet`, `Execute` and `ExecuteContext` retain their trusted extension points. A context deadline alone does not select this profile.

Configure **every** execution, compilation and input allowance with a positive value. Execution, parser and input depths must be between 1 and 64. These are application-selected ceilings, not recommended tenant defaults; choose them from the application's accepted workloads. Steps are implementation work units. Value/input bytes account logical construction, not heap, RSS or allocator capacity.

```go
opts := jet.RestrictedOptions{
    Execution: jet.ExecutionOptions{
        MaxSteps: 1000000, MaxDepth: 64,
        MaxValueBytes: 8 << 20, MaxOutputBytes: 1 << 20,
        ParseOptions: jet.ParseOptions{
            MaxSourceBytes: 64 << 10, MaxTotalSourceBytes: 256 << 10,
            MaxNodes: 20000, MaxSteps: 1000000,
            MaxDepth: 64, MaxTemplates: 32,
        },
    },
    Input: jet.InputOptions{
        MaxBytes: 256 << 10, MaxItems: 10000,
        MaxSteps: 1000000, MaxDepth: 64,
    },
    Text: true,
}
set, err := jet.NewRestrictedSet(ctx, map[string]string{
    "main.jet": `Hello {{ upper(.name) }}`,
}, opts)
if err != nil { return err }
tpl, err := set.GetTemplateContext(ctx, "main.jet", jet.ParseOptions{})
if err != nil { return err }
result, err := tpl.RenderRestrictedContext(ctx, nil, map[string]string{"name": "world"})
```

`RenderRestrictedContext` owns its bounded result buffer and returns an empty string on error. Buffer retention and the final result copy consume the value allowance. Calling a restricted template's `Execute` or `ExecuteContext` with an external writer returns `ErrCapabilityDenied`; a writer callback could otherwise start another render with a fresh allowance. Cancellation is synchronous, and no owned worker survives return. For supported Go versions, render cancellation preserves both the context error and custom cause.

## Accepted values and bindings

| Surface | Contract |
| --- | --- |
| Input data, vars and `RestrictedOptions.Globals` | Method-free booleans, signed/unsigned integers except uintptr, floats, strings, arrays/slices and string-keyed maps. Non-nil interface values are inspected recursively. Typed nil collections, named method-free scalars and arrays preserve their types and existing Jet semantics. |
| Input ownership | Values are copied under shared byte/item/work/depth admission. Strings can share immutable storage. Repeated acyclic children are copied per occurrence and charged each time. Globals are copied at construction and again with vars/data for each render. Template mutations cannot change the caller or the next render. |
| Rejected input | Functions, method-bearing values (including pointer-receiver methods), pointers, structs, channels, complex/unsafe/uintptr values and cyclic graphs. Unused values are still admitted or rejected. Callers must synchronize all mutations during source/input admission; copying cannot make concurrent access to a Go map safe. |
| Default bindings | `lower`, `upper`, `hasPrefix`, `hasSuffix`, `repeat`, `replace`, `split`, `trimSpace`, `html`, `url`, `safeHtml`, `safeJs`, `raw`, `unsafe`, `json`, `writeJson`, `map`, `slice`, `array`, `isset`, `len`, `includeIfExists`, `exec`, `ints`. Their implementation is the metered private registry. `dump` is denied. |
| Aliases | `LookupBuiltin(name)` returns an opaque `BuiltinBinding`; `BuiltinAliases` installs explicit aliases. Identity survives local variables, maps/slices and template return values. Names, Go function addresses, matching signatures and cooperative budget calls grant no authority. There is no API to approve an arbitrary Go callback. |
| Generated values | Data plus the sealed JSON/hidden-boolean renderers and integer ranger. Native collections use internal pooled rangers. Unknown capability roles, AST kinds and unsupported reflective conversions terminate. Generated cycles remain bounded by execution depth/work during traversal. |
| Host hooks | External Func/Go functions, Renderer/Ranger/SafeWriter, Stringer/Error/Formatter and JSON/TextMarshaler hooks are denied before invocation. Public Runtime/Arguments compatibility adapters are available to trusted callbacks, which this profile cannot invoke. |

Ordinary `try/catch err` binds `err` to a plain string in this profile. Print it directly (`{{ err }}`), pass it to string/data builtins, or store it in a map/slice; it has no Go `Error()` method. Interpreter-owned messages are copied under work/value reservations, capped at 512 UTF-8-safe bytes with `...` on truncation (before output escaping). Unknown recovered errors use the fixed text `template execution failed`; constructing this diagnostic never invokes their Error/Stringer hooks. Trusted APIs retain their Go error binding and cause chains. Capability, cancellation and resource failures remain terminal and never bind a catch diagnostic.

`replace` retains Jet's four-argument contract. This profile does not supply application helpers such as three-argument replace, join, formatNumber or time helpers. They need metered owning-layer implementations and explicit registry dispositions before an application can preserve those helper contracts. Registering their original Go functions is not an admission mechanism.

## Compilation, composition and immutability

Sources are admitted into a private in-memory bundle under source/total/template/work limits. Canonical names are at most 1,024 bytes; duplicate canonical names are rejected. Optional missing names must be explicitly listed in `OptionalTemplates`, within the same namespace count cap. Only authorized absence can be hidden by `includeIfExists`. Unauthorized names and resource/cancellation failures terminate through both try/catch and evaluated isset operands; isset retains its existing lazy expression semantics.

Includes, imports, inheritance, yield/content, exec and custom delimiters/comments retain native syntax and share active execution/compilation allowances. Names are canonicalized and authorized before cache access. Cache entries must originate from this Set and are revalidated against each request's compilation allowance. A foreign cached template is ignored. `Parse`/`ParseContext` also accept bounded inline source; they grant no additional loaded namespace. Inline source and loaded dependencies share the render's compilation allowance.

Zero or looser per-call limits cannot weaken a restricted Set's ceilings. Tighter per-call compilation limits remain useful. Set mutation through AddGlobal/AddGlobalFunc or an Option applied after construction panics with `ErrCapabilityDenied` before changing configuration. LookupGlobal returns a fresh data snapshot. Do not mutate exported templates/ASTs or copy/modify Set values after publication; compiled ASTs are immutable under the supported contract.

Use `errors.Is` for `ErrCapabilityDenied`, `ErrInputLimit`, the execution limit errors and [compilation limit errors](compilation.md). Capability/resource failures are terminal and cannot become a successful catch or false result.

This is not OS isolation or an exact CPU/memory guarantee. Trusted APIs still synchronously own arbitrary native calls, writers and loaders and cannot forcibly interrupt them. Separate process isolation is required when the contract includes host-code blocking, process faults or hard memory containment. The restricted profile's accepted builtin cost coverage and the consuming application's helper/workflow validation are separate release gates.
