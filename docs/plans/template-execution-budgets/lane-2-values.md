# Lane 2: allocation, output and builtin costs

**Prepared, not started.** Begin only after the owner's execution instruction. One implementer owns diagnosis, implementation, self-review and checks.

Read [plan](plan.md), [RFC-0002](../../rfcs/0002-value-and-output-budgets.md), [RFC-0004](../../rfcs/0004-restricted-execution.md), [audit](source-audit.md), [PR register](upstream-prs.md), [validation](validation.md), and applicable coding guides.

## Independent first work

1. Inventory every allocation, output sink and builtin call, linking source symbols. Mark exact/proven upper bound, incremental accounting, or restricted denial. Include implicit formatting and reflection conversions, discarded exec output and nested try copies.
2. Prepare bounded failing amplification/transaction fixtures and the operation-cost schedule. Do not run OOM inputs on the host or assume `GOMEMLIMIT` isolates them.
3. Review default helper signatures/semantics and report which Karsto helpers require adapter changes to Lane 4. `formatNumber` precision and replace/split/join growth are concrete cases.

## Dependent implementation

After Lane 1 publishes its foundation SHA, rebase and implement J04/J05 against the canonical runtime budget. Do not create a temporary duplicate meter while waiting.

- Check arithmetic/reserve before allocation. Define cumulative logical charges and separate input admission from produced allocation. Avoid recursive alias-size accounting.
- Meter output after escaping and across all writer paths. Charge try output when generated, avoid double output charge on commit, charge temporary buffer copies as values, retain charges after rollback. A writer error cannot become success.
- Share fuel/output/value budgets with exec even when it discards bytes. Add finite work estimates/checkpoints within data-dependent native operations, not one charge per potentially huge function.
- Preserve valid builtin/helper behavior within limits. `SafeWriter` compatibility requires checked error/terminal handling, not a public signature break. Unsupported reflection/host behavior is an explicit restricted denial supplied through Lane 3's policy.
- Keep builtin descriptors, implementation and size/work bounds together so future additions cannot omit metering. No post-return size check presented as prevention.

## Ownership and acceptance

Own `default.go`, builtin/value/output helpers and corresponding new tests after Lane 1's nested-execution patch. Coordinate `eval.go` edits as a dependent diff. Lane 3 owns restricted policy selection and input conversion; supply the vetted builtin descriptors and denial list it needs. Lane 4 owns Karsto helper registration and final performance report.

- [ ] V01–V08 and relevant R07/R09 pass with matching baseline/fix fixtures.
- [ ] Allocation profiles establish pre-allocation rejection; no giant allocation is made merely to discover the budget failure.
- [ ] Every builtin has a complete capability/work/value contract; unsupported ones fail explicitly in restricted mode.
- [ ] Upstream escaping/template compatibility and normal/race tests pass.
- [ ] PRs describe logical accounting, output generation semantics and arbitrary-writer/callback limitations accurately.

Provide exact SHA, meter API, changed helper contracts, logs/receipts and remaining exclusions. Do not tune away checks to meet the 20% target. Read profiles before attributing overhead. Follow the shared gate and stage-reuse policy; no dedicated fuzzing without approval.
