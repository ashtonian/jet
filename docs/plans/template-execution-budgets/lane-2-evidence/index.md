# Portable lane 2 validation artifacts

These are fork-only evidence for [the handoff](../lane-2-evidence.md). `result.json` records stage command, toolchain, environment, source/dependency manifest, deadline/cleanup, exit and stable-input proof. Identical manifests are stored once under `manifests/`. Raw local artifacts remain under `.git/lane-2-validation/`; `raw-artifact-sha256.json` identifies the unmodified originals, including uncommitted profile/binary artifacts. Text copies replace machine paths/host identifiers and omit other owners' queue notices; trailing whitespace is normalized and control patches use zero context. These are adapted receipts, not byte-identical originals.

The `*-control-final` stages intentionally exit 1 because one reservation is disabled. Their `control.patch` and `control-result.json` identify the sole source mutation and restoration. The corresponding guarded stages pass. `paired-proof.json` verifies identical remaining inputs/configuration/toolchain. Each pair has normal-build flat/cumulative/source profile views and child peak RSS in `run.log`; race runs are separate. Total profile/RSS values include harness/runtime allocations and establish no speed or heap-cap claim.

For the exact paired render inputs, use production commit `5583ec15361aea8f98c7a0d5a1fcbef25d407c0e` and replace `execution_values_test.go` with the committed `profile-execution-values-test.go.txt` snapshot. Its SHA-256 matches the receipts; the final branch adds independent loop and discarded-output/isset cases to this snapshot. Other fixture sources and dependencies are committed at that production SHA.

To repeat finite controls in a disposable checkout with installed host-gate:

1. Place `run.py`, `controlled_profiles.py` and `inspect_final_profiles.py` in `.git/lane-2-validation/`.
2. Run `python3 .git/lane-2-validation/controlled_profiles.py final`. This runs both expected-failing controls under focused admission with 120-second execution and five-second cleanup bounds; the largest produced string is 128 KiB. It restores each modified source even on TERM. Do not run concurrent editors/validators against those files.
3. Run each guarded command from its receipt with the same environment/options and unchanged source. Keep absolute profile/output paths. Both guarded selections must pass before interpreting the controls.
4. Run `python3 .git/lane-2-validation/run.py final-profile-inspection run-check 120 python3 .git/lane-2-validation/inspect_final_profiles.py`. Inspect every profile's flat, cumulative and source views before conclusions.

A Go heap target is not process memory isolation. These finite fixtures need no OOM/stack-overflow attempt or dedicated fuzzing. The original baseline/diagnostic/argument/numeric stages are independent historical observations, not matched performance samples against final CI. Earlier rebase HEADs survive in local backup refs; the source manifests record dirty/untracked inputs rather than treating HEAD alone as the tested source.

`go116-final-platform` preserves the Darwin full-suite result and exact SDK/compiler/link hashes. `go116-final-options` and `go116-final-terminal` cover the final loop/byte-option/discard tests on the same owned SDK. The preceding `go116-final-increment` invocation was rejected before admission due its command shape; it provides no test result. `value-guarded` is the earlier failed profile-write attempt; use the successful `*-guarded-final` artifacts. The Go 1.16 JSON CI failure and corrected passing matrix are linked in the handoff; the final commit's CI is linked from fork PR #2.

No restricted policy, Karsto tenant workflow, release or complete performance acceptance is established by this artifact directory.
