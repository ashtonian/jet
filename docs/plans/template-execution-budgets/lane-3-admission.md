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
- [x] C01–C07 restricted runtime workflows pass, including malicious alias/cycle cases.
- [x] Compilation return joins all owned lexer/read activity.
- [x] Every implicit Go invocation site has a policy disposition and test.
- [x] Restricted/unrestricted mixed concurrent use does not leak policy or environment.
- [x] Supported builtin/loader/input subset is documented; no unrestricted callback isolation claim.

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
| Input and loader preparation | `bd2479e`, `ffc1d8a`, completed in `c0cffa09b8b72b8dd40d9e35d8e50fc7bae3e959`: bounded snapshots, immutable bundles and enforced restricted rendering. [Capability inventory](lane-3-capabilities.md) reviews every invocation disposition. |
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

### Restricted capability handoff

Implementation `c0cffa09b8b72b8dd40d9e35d8e50fc7bae3e959` is consumed with final Lane 2 tests/docs `a81e7dbd5036bebe9b8bcb5de3676c2e87ccd3fe` in canonical integration `2d95fa9e9f02a831febbd62c0fb10995a3496e7b`. The exact builtin production candidate remains `5583ec15361aea8f98c7a0d5a1fcbef25d407c0e`. Runtime foundation `32997fd20e6986220c8c0b60bfff9cedc49d46b7` is included through that dependency, in addition to the original explicit Lane 1 merge. Lane 2's completed [cost evidence](lane-2-evidence.md) supplies the reservation controls, inspected allocation profiles and minimum-Go compatibility; policy denial now completes the Jet-side capability handoff. Its earlier `e883796` JSON bound failure is retained as failure history, not accepted coverage.

Public API: NewRestrictedSet(ctx, sources, RestrictedOptions), InputOptions, opaque BuiltinBinding/LookupBuiltin and Template.RenderRestrictedContext. All execution/parse/input budgets must be positive; depth is at most 64. Set configuration, builtin identities and the in-memory namespace are immutable. Each render snapshots globals/vars/data under shared input admission and returns an owned bounded string, empty on failure. External writer callbacks are denied on restricted templates, closing public reentry through a fresh top-level budget. External native loaders are not admitted. Existing unrestricted callbacks/writers/loaders remain synchronous and compatible.

Supported default bindings, source/data ownership, logical accounting exclusions and denial semantics are in the [public contract](../../restricted.md). The [source-linked capability review](lane-3-capabilities.md) signs off each invocation disposition against tests. Names cannot authorize Go functions; genuine private descriptor aliases survive variables, containers and template returns. Dump, unknown capability/AST/value kinds, native formatting/serialization, channels and unsupported reflection terminate. Capability denial and limits bypass evaluated try/isset recovery; isset retains existing lazy semantics.

| Acceptance | Exact evidence |
| --- | --- |
| Controlled pre-profile baseline | Root Go source overlaid from exact `bda0fa8` (same production as the pre-profile trusted workflow), same `{{ value() }}` source, function input and positive execution/parse budgets. It returns `called`, nil error and called=true, failing the denial assertion. Sources are read from Git into an evidence fixture, not a second checkout/evaluator. |
| Restricted fix | Matching function fixture in TestRestrictedDenialWorkflow returns ErrCapabilityDenied, empty result and called=false. Normal/race named-package checks pass, including all C01–C07 dispositions. Source manifests include tracked/untracked inputs and overlay hashes. |
| Depth ceiling | Externally supervised 10-second child, 256 KiB Go stack, configured depth 64. Recursive yield, deep finite parse and nested input terminate with their distinct depth/input errors; normal/race results are separate. |
| Ownership | TestExecutionOwnsBlockingWriter and CompilationBlockedLoaderOwnership use entry/release/exit synchronization and join the native owner. Restricted external writers/loaders cannot enter that ownership path. |
| Compilation-only review | Exact #6 head `484d70de3c8bfb6f9da06483637e14e276eff439` passes compiler/lexer workflows via an exact-source Go overlay. The compilation review remains independently buildable. |
| Full integrated CI | `2d95fa9e9f02a831febbd62c0fb10995a3496e7b` passes Go 1.16.15 normal, stable normal and stable race: [run 42431462657](https://github.com/ashtonian/jet/actions/runs/42431462657). |
| Static validation | Final go vet passes with installed focused check admission; receipts cover the final peer test additions. Documentation/source links and final review scope are checked separately. |

[Restricted receipts and manifests](lane-3-evidence/restricted-stages.json) retain commands, execution/cleanup bounds, source identities, exit statuses and sanitized logs. The baseline's exit 1 is expected evidence of missing denial. Earlier fixture/build corrections and the rejected Python run-focused wrapper are retained as failed attempts, not passing coverage. Gate-lock admitted direct go test commands with 180-second execution / five-second TERM cleanup; Go test has a 90-second deadline. Queue waiting is outside those execution deadlines. No dedicated fuzzing was added or run.

### Current fork reviews and remaining limits

Reuse existing [#5](https://github.com/ashtonian/jet/pull/5), [#6](https://github.com/ashtonian/jet/pull/6) and [#7](https://github.com/ashtonian/jet/pull/7); no additional PR was opened and no history rewritten or PR merged. #6 advances to `484d70de3c8bfb6f9da06483637e14e276eff439`; #7 consolidates runtime compilation and restricted capabilities at `2858cef45f7fdc9c100fe0294f367fea63c82a15`, against explicit combined base `dd594d8cd6700395b96badc0b49bfc4b6b235e89`. The base consumes final Lane 2 `a81e7db` and #6; [head CI](https://github.com/ashtonian/jet/actions/runs/42431657520) and [base CI](https://github.com/ashtonian/jet/actions/runs/42431657452) pass all three compatibility jobs. Review trees exclude internal assignment/RFC documents and fork packaging. [Ref/signature evidence](lane-3-evidence/restricted-review-refs.json) records exact mappings; all newly authored commits use the effective configured identity/key and are verified locally and by GitHub. The unchanged third-party #236 commit remains unsigned with preserved authorship.

Lane 3's Jet workflow acceptance is complete. Lane 4 still owns fork module packaging, actual Engine/RenderString/webhook/chain/PDF validation and performance acceptance. LookupBuiltin exposes audited Jet defaults only: Karsto's three-argument replace, join, formatNumber, time and other differing helpers need owning-layer bounded implementations/dispositions; arbitrary cooperative Func registration remains denied. Do not silently change helper semantics or enable tenant-safe rendering before those consumer gates. This is an in-process restricted subset, not OS CPU/memory/crash isolation, and does not close Karsto #209.
