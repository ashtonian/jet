# Source audit and evidence provenance

Inspected 2026-11-07. No runtime experiments were executed in this planning session.

## Immutable baselines

| Repository | Observed source |
| --- | --- |
| CloudyKit/jet | v6.3.3 and fetched `master` both `202464510d5b57f0afa9c5563f75babc1616682c` |
| Karsto | cloned main `50f35351fabea91c8bb4f3cbbb80b542171a6d19` |
| Original RFC-0578 | Karsto commit `841cc3d5eb830cc5542163a65e3643612390924d`, read from an existing worktree; not present on the inspected main |
| Historical reproduction | Karsto #209 report at `6134c805a`; numbers below are reported evidence, not rerun here |

Upstream latest release API returned v6.3.3. Branch inventory: master, fix-embed-loader-on-windows, issue-62, pr/95, support/v3, v1.0. No budget implementation was found in the inspected master or title/body search of relevant issues/PRs. This is not a proof that every fork or unpublished branch lacks a solution; repeat targeted search before implementation.

## Upstream work to reuse

- [PR #236](https://github.com/CloudyKit/jet/pull/236), open at `c9ec08935ddf15eaf9328128e008faf0bef5e079`: active dependency-chain cycle detection for imports/extends, path aliases, Set.Parse and concurrent diamond dependencies. Runtime includes remain outside that patch. Source diff inspected.
- [PR #234](https://github.com/CloudyKit/jet/pull/234): reported globals locking/nil data fixes in `dump`. Search result inspected; verify full patch and current disposition before changing dump. Do not attribute its fix to execution budgeting.
- [Issue #117](https://github.com/CloudyKit/jet/issues/117): historical context leak report; current runtime cleanup still needs a field-by-field audit.

## Inspected paths and implications

| File / function | Source observation | Required work |
| --- | --- | --- |
| `exec.go`: Execute | Borrows Runtime from a pool, takes no context, follows extends and executes root | Shared execution entry, synchronous cleanup, context checks including empty templates |
| `eval.go`: Runtime.recover | Clears scope/data context, returns runtime to pool; other fields are not all reset | Clear writer/set/content and all execution state; preserve panic semantics deliberately |
| `eval.go`: executeList / NodeRange | No fuel/cancellation; ranger cleanup happens only after loop | Charge statements/advances, unwind-safe cleanup, reject or handle blocking channels |
| `eval.go`: executeYieldBlock / YieldBlock / content | Nested list execution without yield depth bound | Depth before recursive calls and shared lifetime |
| `eval.go`: executeInclude | Include cap 100,000; loads sibling templates at runtime | Practical depth/fuel and bounded runtime compile/load admission |
| `default.go`: exec / includeIfExists | Nested executeList; exec replaces writer with discard | Same budget, policy, source namespace and output accounting |
| `eval.go`: executeTry | Unbounded bytes.Buffer; catches panics; ignores io.Copy error | Terminal errors bypass catch; bounded transactions; restore all dynamic state |
| `eval.go`: isSet | Recovers panic to false | Must not hide cancellation/resource/policy termination |
| `eval.go`: evalAdditiveExpression | String conversion and fmt formatting precede concatenation | Preflight exact/proven bound before allocating; disallow implicit native hooks in restricted mode |
| `eval.go`: evalPipeCallExpression / evalSafeWriter / Renderer | Go reflection/direct callbacks and implicit printing | Policy by capability identity, invocation metering, no false preemption claim |
| `eval.go`: checkEquality / indirect / resolveIndex | Recursive values, pointer traversal, field/method resolution | Meter recursive work and reject unsupported cycles/capabilities; no unrelated semantic fixes mixed in |
| `eval.go`: escapeeWriter / escapeWriter | Ignores underlying writes or returns `(0,nil)` | Correct count/error propagation with SafeWriter compatibility |
| `node.go`: NodeBase.error | Formats an error as text | Preserve errors.Is/As chains |
| `ranger.go`: pooled rangers | Slice/map/channel state retains references; channel Range receives synchronously | Clear on every path; restricted channels/custom rangers denied |
| `default.go`, `dump.go` | Native string, JSON, map/slice constructors and formatting | Full builtin cost/capability inventory |
| `set.go`, `parse.go` | Loader reads and recursive dependency parse occur before execution | Per-operation parse admission, source/node/dependency caps and bounded readers |
| `lex.go`: run / emit / drain | Producer goroutine, unbuffered token sends, parser drains on error | Cancellation must stop and join scanning, or replace with pull lexer |
| `cache.go`, Set globals/cache | Shared parsed templates and mutable globals | No per-render state in cache; policy identity and per-render bindings |

## Downstream workflow

`public-pkg/template/engine.go` starts `tpl.Execute` in a goroutine, returns on ctx.Done, and uses a panic-based writer because Jet's escaping swallows returned errors. The wrapper's source explicitly acknowledges silent-loop continuation. `public-pkg/expr/runtime.go` is a useful convention: evaluate synchronously under deadline/admission, hold a permit until the backend returns, and preserve cancellation causes. Reuse the lifecycle idea, not expr's backend-specific safety claims or a second interpreter wrapper.

`karsto/internal/services/rules/chain/template.go` returns raw source on rendering error and when no Renderer exists. The resolver returns string only, so fail-closed behavior requires changing its internal signature and all callers, not returning an empty string. Callers include action, enrich, external, filter and transform nodes. Several still construct `${...}` paths; preserve that finding separately and test any necessary semantic correction rather than equating it to budget failure.

`forge/webhooks/delivery.go` renders body/headers. `forge/reporting/encoder/pdf/pdf.go` renders HTML before launching an external PDF tool. Verify these workflows with side-effect assertions: no HTTP delivery, action dispatch or PDF subprocess after a rendering error. The external PDF tool's own resource/network/file safety remains a separate contract.

## Reported reproductions from #209

- Non-writing two-billion-iteration loop: deadline returned after 261,380 iterations, followed by 1,077,259 more iterations over the next 200 ms.
- Twenty-six `s = s + s` operations from `"ab"`: 128 MiB intermediate, nine output bytes.
- `{{ block b() }}{{ yield b() }}{{ end }}`: fatal stack overflow.

These are the parent report's observations. Future work must reproduce controlled bounded baselines under matching fixed inputs, then demonstrate the corrected actual renderer workflow. Do not run the raw crash/OOM probes in a shared test process.

## Corrections to the original RFC

“No input can crash the process,” “all defaults are bounded,” “every path is metered,” and “per-node checks are cheap” are desired outcomes or hypotheses, not established facts. Callback return-size checks cannot prevent allocation. Include-only depth misses expressions and yields. Source bytes alone do not bound nested parser recursion. A cache hit can bypass policy. The RFC suite makes these explicit and gates tenant adoption on evidence.
