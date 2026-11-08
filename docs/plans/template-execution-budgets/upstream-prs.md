# Implementation work register

Work breakdown, not a one-PR-per-row submission schedule. Follow the [current publication policy](plan.md#current-publication-policy): fork-only development, no new upstream submissions, internal plans/RFCs excluded, and related work consolidated into existing fork PRs. Preserve upstream conventions and meaningful tests; do not expose unsupported budget fields.

| ID | Suggested title | Owner | Base / predecessor | Scope and evidence | Exclude |
| --- | --- | --- | --- | --- | --- |
| J01 | Preserve writer error causes / clear pooled execution references | 1 | Pinned upstream, refresh before submission | Separate independent correctness PRs if appropriate; text, escaped, raw and try sink failures; error identity; success/error pool cleanup | New sandbox API, unrelated formatting, dump fixes already covered by #234 |
| J02 | Add synchronous context-aware template execution | 1 | Necessary J01 patches | Additive ExecuteContext; native checkpoints; terminal cancellation through try and isset; no post-return work | Memory guarantees and arbitrary callback preemption |
| J03 | Bound interpreter steps and recursive execution | 1 | J02 | One execution budget across list/expression/value traversal and nested composition; fatal recursion subprocess regression | Policy/default changes for existing Execute |
| J04 | Account for value growth and buffered output | 2 | J03 | Checked reservations, amplification guard, nested try transaction/output semantics, discarded exec output | Application helpers, allocator/RSS claims |
| J05 | Meter allocation and work in template builtins | 2 | J04 | Complete builtin inventory and bounded implementations; exact compatibility fixtures | Silent helper removals, user callbacks approved by name alone |
| J06 | Consume cyclic dependency detection | 3 | Re-check #236 | Prefer upstream merge; otherwise preserve original authorship/provenance and add only demonstrably missing coverage | Duplicate competing cycle PR |
| J07 | Add bounded context-aware template compilation | 3 | J06; runtime bridge after J03 | Source/node/depth/dependency budgets, lexer lifecycle, loaders/cache policy | AST wholesale refactor unrelated to limits |
| J08 | Add an explicit restricted execution policy | 3 | J03, J05, J07 | Enforced input/capability admission, immutable bindings, cache namespace, no reflection escape routes | Karsto trust decisions and E-codes |
| J09 | Document execution cost and compatibility results | 4 | Relevant exact candidate | Neutral fixtures/harness plus truthful baseline comparisons; can be included in owning PRs | Local machine paths and unmeasured safety/performance claims |
| F01 | Publish fork module identity | 4 | Accepted fork candidate | Fork-only module/self-import/example rewrite and external consumer resolution | Any upstream submission |
| K01 | Fail chain actions closed on template errors | 4 | Current Karsto | `(string,error)` resolver; all callers propagate before effects; nil renderer denial on dynamic input | Jet changes, raw-source or empty-result fallback |
| K02 | Adopt metered Jet through the Renderer abstraction | 4 | J03–J08 and pinned F01 artifact; K01 | Synchronous wrapper, supported helpers, validated defaults, error mapping/cache and real consumers | Claims of arbitrary Go isolation |
| K03 | Verify tenant template execution end to end | 4 | K02 | Workflow tests, benchmark/profile report and #209 acceptance evidence | Closing #209 with unit-only evidence |

J02/J03 may combine if separating them weakens the public contract. J04/J05 may remain separate review steps, but neither individually satisfies restricted execution. Upstream can accept generic bug fixes or execution APIs without endorsing the fork's full policy. Record maintainer feedback as explicit RFC revisions rather than hiding a contract fork.

## PR description outline

1. Concrete failing behavior and minimal bounded reproducer.
2. Resulting behavior, public API additions and compatibility effects.
3. Where enforcement occurs and what it cannot cover.
4. Exact baseline/fix commands, versions and results; normal versus race separated.
5. Benchmarks and profile explanation when relevant; links to direct prerequisites.

No internal assignment history, generated-by text or assistant credits. Do not substitute a large checklist for explaining the change. Keep a concise coverage table where the security boundary benefits from it. Passing implementation and validation gates does not authorize upstream submission; the publication hold remains in effect.
