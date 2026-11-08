# Upstream-oriented PR register

Planning register, not opened implementation PRs. Preserve upstream conventions and a small review surface. Each row may split only when both resulting patches compile, have meaningful tests and explain their own behavior. Do not create a test-only red PR or expose unsupported budget fields.

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

No internal assignment history, generated-by text or assistant credits. Do not substitute a large checklist for explaining the change. Keep a concise coverage table where the security boundary benefits from it. Open upstream submissions only after their code and evidence exist; this planning session creates no upstream issue/PR or maintainer notification.

## Lane 3 fork review state (2026-11-07)

The owner has superseded upstream-submission planning: **use ashtonian/jet only; do not open new upstream PRs**. Existing third-party #236 was consumed unchanged, not resubmitted.

| Scope | Fork PR | Exact review head | Exact base |
| --- | --- | --- | --- |
| Pull scanner | [#5](https://github.com/ashtonian/jet/pull/5) | `6cf13002e0a9fcf0548b8a426c428802c7890553` | `c9ec08935ddf15eaf9328128e008faf0bef5e079` |
| Compilation admission | [#6](https://github.com/ashtonian/jet/pull/6) | `5e5aa42662d9f6fc5372b7809cd03c6ddbb48486` | `6cf13002e0a9fcf0548b8a426c428802c7890553` |
| Shared runtime compilation | [#7](https://github.com/ashtonian/jet/pull/7) | `98f45c6b1d87b9fb8941291c94d51441c3a5b4b9` | `c422a0215ed40cc17d2282688d3626a1014ac7d7` |

The combined runtime base includes explicit foundation `d2f089ce4f617dd512c7eee6b5d825f2c0b57914`, compiler #6 and the compatibility workflow. Normal/race/minimum-Go CI passes on the canonical and clean review candidates; [lane handoff and receipts](lane-3-admission.md) provide controlled baselines, tested inputs and remaining gates. The reviews exclude internal task documents, private admission preparation and fork packaging. No PR was merged. Restricted selection remains disabled pending J05's accepted cost/binding implementation and C01–C07 enforcement.
