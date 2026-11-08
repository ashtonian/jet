# Lane 3: bounded compilation and restricted capabilities

**Execution started 2026-11-07 on the owner's instruction.** One implementer performs diagnosis, code, self-review and validation.

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

- [x] P01–P07 compilation workflows pass with controlled baseline/fix evidence.
- [ ] C01–C07 restricted runtime workflows pass, including malicious alias/cycle cases.
- [x] Compilation return joins all owned lexer/read activity.
- [ ] Every implicit Go invocation site has a policy disposition and test.
- [ ] Restricted/unrestricted mixed concurrent use does not leak policy or environment.
- [ ] Supported builtin/loader/input subset is documented; no unrestricted callback isolation claim.

Handoff exact SHAs, policy/input types, compile-cache metadata/key rules, builtin registry contract and result receipts. Lane 4 cannot enable tenant-safe rendering before this acceptance and Lane 2's cost coverage pass. Do not launch agents, run dedicated fuzzing, or bypass shared host admission.

## Execution handoff (2026-11-07)

The owner started execution and subsequently directed **fork-only work: no new upstream PRs**. All new review PRs are in `ashtonian/jet`; none were merged. Only this checkout was edited, with one implementer and no additional sessions/agents.

Read the canonical global AGENTS/CLAUDE files and all linked Jet RFCs. This clone and its ancestors have no repository CLAUDE, `.claude/rules`, Go-style/review-loop rules or lint configuration; none were copied from another project. The module and implementation retain Go 1.16 compatibility.

### Delivered compilation and preparation

| Contract | Canonical commits / evidence |
| --- | --- |
| Cycle detection | Original #236 `c9ec08935ddf15eaf9328128e008faf0bef5e079`, unchanged with original author, consumed in signed merge `f7f7e22`. Eight-read bounded baseline fails the cycle regressions; fix and concurrent diamonds pass. |
| Synchronous scanner | `4d419fd`, tightened fixture `415d548`. Parser error `{{ if }}` followed by text drains the baseline source; the pull lexer stops at the error. An earlier `{{ ) }}` probe only tested a lexer that had already stopped and is not causal evidence. |
| Bounded compilation | `9a8d22e`, cache-parity correction `a91c96b`, terminal cleanup `195312a`, native-panic compatibility `d39d1d5`. Public ParseContext/GetTemplateContext, source/total bytes, nodes, work, depth and template-count limits; [API documentation](../../compilation.md). |
| Runtime compilation bridge | `fd1bc81`, based on runtime `d2f089ce4f617dd512c7eee6b5d825f2c0b57914`, consumed at `1c63e78`. ExecutionOptions.ParseOptions admits the initial graph and shares one allowance through include/exec/includeIfExists. Fresh-per-load controlled baseline bypasses total source limits; fixed normal/race workflows reject terminally. |
| Input and loader preparation | `bd2479e`, `ffc1d8a`: private data snapshots and immutable bounded template bundles. No restricted selection/API is enabled. [Capability inventory](lane-3-capabilities.md) enumerates remaining runtime dispositions. |
| Compatibility corpus/CI | Consumed Lane 4 `accce48cdc3d7813eba528c8b90d10697e202f02` at `9432a63`. No fork module/import rewrite. |

Admission is operation-local. Context, active ancestors, counters, lookahead/source and expression-depth bookkeeping are cleared before caching. Immutable source/node/depth metadata stays with ASTs. Source bytes/template count use canonical names; repeated versions charge growth; development reparsing charges every construction. Cached ASTs charge their nodes once per operation, including trees first constructed during that operation. Every traversal still consumes work. Cache entries must originate from the same Set, and stricter requests revalidate the graph. Do not mutate exported ASTs after compilation.

The scanner is synchronous. All loader/reader/close activity finishes before return; arbitrary native loaders may still block. ContextLoader is cooperative, not forced interruption. First compilation cancellation/resource failure remains sticky through cleanup, including a Close error-panic, while actual runtime faults preserve panic behavior. No detached workers or sandbox/heap/RSS guarantee is claimed.

### Validation

- Full local normal and race suites pass on the independent compiler/input batch; the exact source/untracked/dependency/toolchain manifests and bounded stage receipts are in [lane-3-evidence](lane-3-evidence/stage-receipts.json).
- Integrated code `195312aae4023f01a3d9ec84e3e12281edcbbe15` passes [CI](https://github.com/ashtonian/jet/actions/runs/42426896557): Go 1.16.15 normal, stable normal and stable race. Earlier integrated runs are preserved in the receipts.
- Matching 2,000-level parentheses, list and finite-dependency inputs overflow 256 KiB child stacks on the baseline. With depth 16 they return ErrParseDepthLimit under the same 10-second external deadline. Normal and race results are separate.
- Cache-node regression uses the same diamond and node cap with cold/warm caches. Before `a91c96b`, cold fails while warm passes; both pass after the fix.
- Cleanup regression uses the same failing source cap and Close panic. The baseline returns successful catch output; `195312a` returns the original terminal limit. One baseline gate release returned 125 after its test exited 1. Its exact admission nonce was recovered through installed gate-lock; no live owner's files were removed. Gate recovery is recorded separately from the test failure.
- Named-package checks use focused admission; full local suites used whole-host admission; vet uses the check lane. All stages have execution/cleanup deadlines and durable logs. No dedicated fuzzing was run.

### Review PRs (fork only)

- [#5](https://github.com/ashtonian/jet/pull/5): pull lexer, base `c9ec08935ddf15eaf9328128e008faf0bef5e079`.
- [#6](https://github.com/ashtonian/jet/pull/6): compilation, base #5. Clean review commits exclude internal assignment documents, input/bundle preparation and fork packaging.
- [#7](https://github.com/ashtonian/jet/pull/7): runtime bridge on the explicit combined runtime/compilation review base. [Ref mapping](lane-3-evidence/fork-review-refs.json) records canonical/review SHAs and exact prerequisites.

### Remaining capability gate

Reviewed Lane 2 candidate `e883796c80058ad4f0c92458d6b2e6e64e9e7ec9` by immutable fetch (its task branch was rebased, so the earlier tracking ref is not authoritative). Its private immutable builtin descriptor registry supplies the needed identity contract. The original [Go 1.16 CI fails](https://github.com/ashtonian/jet/actions/runs/42426825673/job/128566779953): TestEncodedValueSizes reports JSON bound 27 versus 35 produced. The corrected candidate `5583ec15361aea8f98c7a0d5a1fcbef25d407c0e` now passes [all CI jobs](https://github.com/ashtonian/jet/actions/runs/42428373277). It is ready for dependent capability implementation; final cost/profile acceptance and tenant adoption remain gated.

Next consume the corrected J05 SHA against its recorded contracts and implement immutable restricted selection, runtime capability checks, canonical authorization before cache, per-render global snapshots, and the finite audited helper surface needed by Lane 4. Do not approve host callbacks by names/program counters or infer safety from deadlines. Private input/bundle preparation is not J08 enforcement. C01–C07, mixed restricted/unrestricted concurrency and tenant adoption remain unchecked; Lane 4 must not enable tenant-safe rendering from this handoff.

The current publication policy requires reusing existing fork PRs and preserving their history. Further capability work will consolidate in the existing review sequence rather than create a PR per work ID.
