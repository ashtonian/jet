# Lane 4: Karsto adoption, workflows and performance

**Prepared, not started.** Begin only after the owner's execution instruction. One implementer handles this lane end to end. Read both repositories' current guides and queue ownership before edits.

Read [plan](plan.md), [RFC-0005](../../rfcs/0005-fork-and-upstream-strategy.md), [validation matrix](validation.md), [PR register](upstream-prs.md), Karsto RFC-0578 and its linked integration plan. Source baseline for this planning batch: Karsto `50f35351f`; refresh before implementation and preserve unrelated work.

## Independent work

1. Reproduce the actual public Engine workflow under bounded controlled inputs. Establish wrapper return/callback lifetime separately from native silent-loop computation. Do not certify the renderer from Jet-only tests.
2. Enumerate every `Renderer`, constructor, `RegisterFunc`, `Funcs`, named/inline loader and cache consumer. Classify tenant-authored versus reviewed process-owned templates. Inventory current default helper names and behavior.
3. Implement K01 independently: change the internal chain resolver to return `(string,error)` and update every action/enrich/external/filter/transform caller. Resolve all fields before side effects or message mutation; dynamic input with nil Renderer errors. Static strings remain literals. No raw-source or empty-string fallback.
4. Prepare fixtures and baseline benchmark/profile harness for upstream Jet and current Karsto. Keep normal builds separate from race/instrumented results; fixture sizes and options must match future comparisons.

## Dependent adoption

After all Jet protection contracts pass, implement F01 packaging and K02 against a pushed immutable fork version. Do not commit a local replace or point a released library at an unpushed task SHA. Keep packaging out of upstream PR diffs.

Preserve Renderer, Render/RenderString, NewHTMLEngine/NewTextEngine, named/inline templates and compile-cache behavior. Remove the abandoned-worker implementation, use synchronous ExecuteContext, and enforce bounded parse/input/capability admission through the same canonical engine. Use the earlier caller/configured deadline and keep ownership until execution returns.

Karsto supplies validated positive defaults and cause-preserving error mapping; Jet stays application-independent. Reuse expr's synchronous lifecycle/concurrency conventions where appropriate, without importing its language-specific budget assumptions. Do not silently change custom callback trust: restricted tenant engines reject arbitrary callbacks; reviewed trusted consumers select an explicit compatible contract.

Adapt every supported helper to a metered implementation, preserving names/results under budget. Errors expose timeout/cancel, step, depth, output, value, input/source and policy failure distinctly with errors.Is. Coordinate allocated E-code range and generated catalogs under existing ownership; do not hand-edit generated catalogs or reuse occupied codes.

## Actual workflows and final evidence

- Webhook body/header error: no HTTP delivery.
- Rule-chain error: node fails before dispatch/mutation; error follows the chain failure path.
- PDF HTML error: external PDF process never launches. The PDF binary's own resource/network safety is separate.
- Named/inline, HTML/text, helper, cache and concurrency scenarios preserve supported semantics.
- External fork-module consumer resolves with no replace directive or dual Jet type ambiguity.

Complete I01–I08 and coordinate exact-SHA normal/race upstream compatibility, then collect the validation matrix's performance suite. Own the final report, not other lanes' implementation or self-review. Inspect CPU/alloc/heap/mutex/block profiles before explaining numbers. Report actual p50/p99, ns/op, B/op, allocs/op, cancellation overshoot and sustained CPU; never weaken limits to meet 20%.

## Done means

- [ ] Real consumer acceptance passes through the canonical Renderer, with no orphaned work.
- [ ] All helper/capability/parse/cache boundaries and intentional compatibility changes documented.
- [ ] Fork packaging is independently resolvable and removable from upstream submission diffs.
- [ ] Performance/compatibility report links immutable inputs and durable results; failures/skips remain explicit.
- [ ] RFC-0578, relevant API spec/README/error docs and #209 updated only with measured results and remaining limits.

Do not close #209, merge incomplete tenant protection, publish a tenant-ready release, or submit untested upstream code. Follow host admission and bounded stage supervision, and the repositories' signed-commit/CI rules. Update PR descriptions rather than repetitive status comments.
