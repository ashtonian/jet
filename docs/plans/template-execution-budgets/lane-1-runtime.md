# Lane 1: runtime ownership and metering

**Prepared, not started.** Begin only after the owner's execution instruction. Use one local implementer for diagnosis, code, self-review and verification; do not spawn reviewers or subagents.

## Read first

[Plan and shared-file ownership](plan.md), [source audit](source-audit.md), [RFC-0001](../../rfcs/0001-execution-lifecycle.md), [RFC-0005](../../rfcs/0005-fork-and-upstream-strategy.md), [runtime validation](validation.md), and current repository/global coding guides. Refresh upstream master and relevant PRs; baseline v6.3.3 is `202464510d5b57f0afa9c5563f75babc1616682c`.

## Assignment

Deliver J01–J03 as compilable, upstream-reviewable PRs. Own the execution budget and terminal failure mechanism; no other lane builds an alternate one. Preserve `Execute`, syntax and unrestricted extension points. Public `ExecuteContext` remains synchronous; no timeout worker/select.

1. Reproduce self-yield failure in a bounded subprocess, silent computation, ignored sink errors and retained runtime references. Keep independent causes distinct.
2. Repair cause wrapping, relevant writer failures and pooled-state cleanup as independent correctness patches where separable. Inspect every recover and resource acquisition, including rangers and try scope restoration.
3. Implement context checkpoints and sticky terminal unwind through both try and isset. Handle entry, completion, empty templates and call-return checkpoints. Specify panic compatibility without hiding programming faults as budget errors.
4. Implement fuel/depth across statements, loop advances, recursive expressions/value comparison, blocks/content/yield/include/exec and inheritance traversal. A single budget belongs to one render. Bounded values and capabilities are later protections, not claims this PR makes.
5. Publish the internal metering/reservation/terminal API for lanes 2/3. Do not expose unenforced value/output fields yet. Context should flow explicitly; any execution-local adaptation to existing callback signatures must be documented and cleared.
6. Run your own R01–R12 and upstream compatibility/race coverage, review performance profiles for introduced dispatch costs, and update the RFC/PR with actual exclusions.

## Handoff contract

Lane 2 needs checked remaining-budget reservation, fatal error propagation, a try/output transaction hook, and nested-call state. Lane 3 needs policy checks before native invocation, a compilation-context bridge and a terminal denial path. Agree those contracts in code/godoc and test them; an invented placeholder interface is not completion.

Shared evaluator changes from other lanes wait for the recorded foundation SHA, then arrive as small dependent PRs. Send only concrete contract/patch references; do not delegate your self-review. Preserve upstream fixes and third-party authorship.

## Done means

- [ ] Controlled baseline and fixed interpreter cases pass under matching inputs, with externally bounded dangerous tests.
- [ ] No owned interpreter work or pooled caller references after return on success/error/cancel/panic.
- [ ] All native loop/recursion entry points have a metering disposition; try/isset cannot swallow termination.
- [ ] Legacy compatibility and positive-limit API tests pass, causes remain inspectable, race results recorded separately.
- [ ] J01–J03 descriptions state that values, compilation and arbitrary native callbacks remain outside this milestone.

Do not merge a tenant-safe claim or close Karsto #209 from this lane's results. Use the shared host gate/deadline/log rules in the plan; prefer CI for full validation. Preserve scope while a stage runs and resume only invalidated/failed checks.
