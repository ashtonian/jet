# Lane 2: allocation, output and builtin costs

**Execution started 2026-11-07** under the owner's explicit lane-start instruction, superseding the prepared planning hold. One implementer owns diagnosis, implementation, self-review and checks.

Read [plan](plan.md), [RFC-0002](../../rfcs/0002-value-and-output-budgets.md), [RFC-0004](../../rfcs/0004-restricted-execution.md), [audit](source-audit.md), [PR register](upstream-prs.md), [validation](validation.md), and applicable coding guides.

## Independent first work

1. Inventory every allocation, output sink and builtin call, linking source symbols. Mark exact/proven upper bound, incremental accounting, or restricted denial. Include implicit formatting and reflection conversions, discarded exec output and nested try copies.
2. Prepare bounded failing amplification/transaction fixtures and the operation-cost schedule. Do not run OOM inputs on the host or assume `GOMEMLIMIT` isolates them.
3. Review default helper signatures/semantics and report which Karsto helpers require adapter changes to Lane 4. `formatNumber` precision and replace/split/join growth are concrete cases.

The source-linked [cost/capability inventory and helper handoff](lane-2-cost-inventory.md) covers all default bindings, implicit formatting/conversion, transactions and discard sinks. [Evidence](lane-2-evidence.md) records bounded baseline fixtures, profiles and current dependency gates. Production candidate `5583ec15361aea8f98c7a0d5a1fcbef25d407c0e` consumes Lane 1 foundation `32997fd20e6986220c8c0b60bfff9cedc49d46b7`; the consolidated final test/docs commit and CI are in [fork draft PR #2](https://github.com/ashtonian/jet/pull/2). No upstream submissions are authorized.

## Dependent implementation

After Lane 1 publishes its foundation SHA, rebase and implement J04/J05 against the canonical runtime budget. Do not create a temporary duplicate meter while waiting.

- Check arithmetic/reserve before allocation. Define cumulative logical charges and separate input admission from produced allocation. Avoid recursive alias-size accounting.
- Meter output after escaping and across all writer paths. Charge try output when generated, avoid double output charge on commit, charge temporary buffer copies as values, retain charges after rollback. A writer error cannot become success.
- Share fuel/output/value budgets with exec even when it discards bytes. Add finite work estimates/checkpoints within data-dependent native operations, not one charge per potentially huge function.
- Preserve valid builtin/helper behavior within limits. `SafeWriter` compatibility requires checked error/terminal handling, not a public signature break. Unsupported reflection/host behavior is an explicit restricted denial supplied through Lane 3's policy.
- Keep builtin descriptors, implementation and size/work bounds together so future additions cannot omit metering. No post-return size check presented as prevention.

## Ownership and acceptance

Own `default.go`, builtin/value/output helpers and corresponding new tests after Lane 1's nested-execution patch. Coordinate `eval.go` edits as a dependent diff. Lane 3 owns restricted policy selection and input conversion; supply the vetted builtin descriptors and denial list it needs. Lane 4 owns Karsto helper registration and final performance report.

- [x] V01–V07, supported-default V08 and value/output R07/R09/R12 paths pass with controlled baseline/fix and workflow fixtures.
- [ ] V08 unsupported-capability denial and policy-error R07 acceptance pass after Lane 3 integration.
- [x] Allocation profiles establish pre-allocation rejection; no giant allocation is made merely to discover the budget failure.
- [ ] Every builtin has a complete capability/work/value contract; unsupported ones fail explicitly in restricted mode.
- [x] Upstream escaping/template compatibility and normal/race tests pass.
- [x] Fork PR/public docs describe logical accounting, output generation semantics and arbitrary-writer/callback limitations accurately.

Provide exact SHA, meter API, changed helper contracts, logs/receipts and remaining exclusions. Do not tune away checks to meet the 20% target. Read profiles before attributing overhead. Follow the shared gate and stage-reuse policy; no dedicated fuzzing without approval.

Lane 2 implementation and its independent checks are complete. The complete builtin schedule and private identity handoff are supplied; the unchecked restricted-policy row requires Lane 3, not a second evaluator or meter. Karsto adapters/workflows, fork release and final performance acceptance remain Lane 4. Keep the existing fork PR draft until dependent gates are demonstrated. Portable evidence and exact remaining limits are linked above.
