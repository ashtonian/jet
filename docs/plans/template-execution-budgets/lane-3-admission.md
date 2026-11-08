# Lane 3: bounded compilation and restricted capabilities

**Prepared, not started.** Begin only on the owner's execution instruction. One implementer performs diagnosis, code, self-review and validation.

Read [plan](plan.md), [RFC-0003](../../rfcs/0003-bounded-compilation.md), [RFC-0004](../../rfcs/0004-restricted-execution.md), [audit](source-audit.md), [validation](validation.md), [PR register](upstream-prs.md) and applicable coding guides.

## Independent first work: compilation

1. Refresh upstream #236 (`c9ec08935ddf15eaf9328128e008faf0bef5e079` when inspected). Prefer consuming the existing fix. Preserve authorship and demonstrate any additional gap before extending it. #234 is a separate dump issue, not a parser prerequisite.
2. Reproduce cyclic/deep-acyclic dependency and deep expression/list parsing under external bounds. Inspect lexer run/send/drain and loader read ownership. Source-byte limits alone do not bound stack depth.
3. Implement J06/J07: context-aware parse/load admission, checked per-source/total/node/depth/template limits, canonical namespace resolution and full lexer/reader cleanup. Keep custom delimiters, valid diamonds and legacy APIs.
4. Make cache hits obey current policy without attaching mutable per-request state to templates/sets. Preserve useful immutable AST reuse across renders. A failed parse cannot poison another caller's active-chain state.

## Dependent work: capabilities

Prepare the capability inventory and data-admission fixtures while Lane 1 builds runtime hooks and Lane 2 builds builtin accounting. Then implement J08 using those hooks/bindings:

- Select restricted policy explicitly and immutably; require positive protection budgets. Do not infer safe mode from a context deadline.
- Admit a bounded data-only graph; deny functions/methods/channels/cycles and implicit Stringer/Marshaler/Renderer/Ranger/SafeWriter hooks unless they are explicitly vetted internal bindings.
- Authorize by capability identity. Function aliases, globals, shadowing and nested return values must not bypass policy. Document concurrency requirements for caller-owned input during admission.
- Preserve template composition through authorized bounded template bundles/loaders. Reject uninterruptible external loaders from the safe contract; never abandon their goroutines.
- Define the internal helper adapter contract with Lane 2 and expose only the surface Lane 4 needs. Do not silently trust a registered Go callback because its name appears on an allowlist.

## Review boundaries

Own `parse.go`, `lex.go`, `set.go`, loading/policy/admission helpers and dedicated tests. Runtime call-site patches are dependent changes after Lane 1; builtin implementations belong to Lane 2. Avoid unrelated parser/AST rewrites. If a pull lexer is chosen, make its compatibility proof a focused PR, not an incidental rewrite hidden in limits.

- [ ] P01–P07 and C01–C07 pass, including cache warm/cold parity and malicious alias/cycle cases.
- [ ] Compilation return joins all owned lexer/read activity.
- [ ] Every implicit Go invocation site has a policy disposition and test.
- [ ] Restricted/unrestricted mixed concurrent use does not leak policy or environment.
- [ ] Supported builtin/loader/input subset is documented; no unrestricted callback isolation claim.

Handoff exact SHAs, policy/input types, compile-cache metadata/key rules, builtin registry contract and result receipts. Lane 4 cannot enable tenant-safe rendering before this acceptance and Lane 2's cost coverage pass. Do not launch agents, run dedicated fuzzing, or bypass shared host admission.
