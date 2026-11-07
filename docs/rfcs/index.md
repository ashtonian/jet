# Template execution budget RFCs

Status: **planning only; implementation has not started**. Updated 2026-11-07.

The owner requested an independent Jet v6 fork, then explicitly directed that this session prepare RFCs and instructions for four future implementers. These documents authorize no automatic agent launch. Proposed APIs are not available in this branch.

| RFC | Decision | Primary implementation owner |
| --- | --- | --- |
| [0001](0001-execution-lifecycle.md) | Synchronous context-aware execution, fuel, depth, terminal errors and cleanup | Lane 1 |
| [0002](0002-value-and-output-budgets.md) | Pre-allocation value accounting, output accounting and metered builtins | Lane 2 |
| [0003](0003-bounded-compilation.md) | Context-aware compilation, source/AST/dependency limits and loader ownership | Lane 3 |
| [0004](0004-restricted-execution.md) | Explicit capability boundary for untrusted templates | Lane 3, with runtime hooks from Lane 1 |
| [0005](0005-fork-and-upstream-strategy.md) | Fork distribution, compatibility, upstream PR decomposition and release gates | Lane 4 |

Start with the [execution plan](../plans/template-execution-budgets/plan.md), [source audit](../plans/template-execution-budgets/source-audit.md), and [validation matrix](../plans/template-execution-budgets/validation.md). Each lane has a standalone assignment linked from the plan.

The motivating downstream contract is [Karsto #209](https://github.com/ashtonian/karsto/issues/209), expanded by Karsto RFC-0578. Jet RFCs describe reusable library behavior; Karsto error codes, helper policies, defaults and consumer changes belong in Karsto.

## Decision status

The owner selected an independently maintained fork and preparation for four implementers. API spellings and numeric defaults below are proposals, pending implementation evidence and upstream review. The initial context/fuel/depth patch is useful on its own but **does not establish an untrusted-template sandbox**. Restricted execution requires all four protection RFCs and downstream acceptance.

No runtime tests, hostile probes, benchmarks or performance measurements were run during this planning session. The source audit distinguishes inspected code from previously reported reproductions.
