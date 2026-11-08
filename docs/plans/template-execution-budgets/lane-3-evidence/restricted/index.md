# Restricted workflow evidence

[Stages](../restricted-stages.json) distinguish the expected failing pre-profile control, passing normal/race checks, standalone compiler review, vet, CI and failed fixture/build attempts. Inputs include untracked Go fixtures, module/dependency inputs, toolchain, gate-granted GOMAXPROCS and overlay replacement hashes. Local metadata uses `$CHECKOUT`, `$HOME`, `$GOTMP` and `$HOST` placeholders. Raw originals remain in `.git/lane3-evidence`; [hashes](../restricted-artifact-hashes.json) cover raw and published files.

The control reads root Go source from exact bda0fa8 into an artifact directory and overlays it for Go compilation. It does not create another checkout or evaluator. Both workflows use `{{ value() }}`, the same finite function input and positive execution/parse settings. The trusted control invokes the function and fails the denial assertion; selecting the profile denies before invocation. Input admission limits apply only in the restricted profile. No race-versus-normal inference is used to establish causality.

Run from the repository root with installed gate-lock. Preparation is read-only Git plus artifact creation; the runner then queues a direct focused go test with a 180-second execution deadline and five-second TERM cleanup. Queue time is separate from execution admission. No heavy full suite or fuzzing is started by these commands.

```sh
python3 docs/plans/template-execution-budgets/lane-3-evidence/restricted/prepare_restricted_control.py baseline
python3 docs/plans/template-execution-budgets/lane-3-evidence/restricted/focused_validation.py reproduced-baseline go test -p=1 -parallel=2 -count=1 -timeout=90s -overlay=.git/lane3-evidence/baseline-overlay.json -run '^TestRestrictedDenialWorkflow$' .
# Expected exit 1: result="called", err=nil, called=true.
python3 docs/plans/template-execution-budgets/lane-3-evidence/restricted/focused_validation.py reproduced-fix go test -p=1 -parallel=2 -count=1 -timeout=90s -run '^TestRestricted' .
# Expected exit 0. Add -race after `go test` for the separate race check.
```

For the independently buildable compiler review, prepare `compiler` instead and pass `-overlay=.git/lane3-evidence/compiler-overlay.json -run '^(TestCompilation|TestLexer|TestLex)'`. It uses exact review head 484d70d. Full Go 1.16/stable/race validation is recorded from CI for canonical 2d95fa9 and the clean review head/base. The restricted depth probe uses an externally bounded child; no giant allocation or dedicated fuzzing was performed.
