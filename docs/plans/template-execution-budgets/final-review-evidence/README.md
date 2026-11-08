# Final review evidence

Evidence for the [2026-11-08 review](../final-review.md). This is an internal fork artifact, excluded from upstream submissions.

- `results.json`: exact source revisions, all six normal render samples, medians and per-fixture ratios.
- `benchstat.txt`: statistical comparisons. Median-of-ratios in the review differs from the reported geomean.
- `profiles.txt`: seven separate profile runs, each with flat/cumulative CPU, alloc_space, inuse_space, mutex, block and `/usr/bin/time -l` RSS output. Profile timings are not benchmark samples.
- `compile-results.txt`: setup/initial-parse samples; restricted setup additionally admits sources/globals and constructs its policy.
- `probes.log`: inherited-panic controls and the final normal/race reproductions. Exit 1 is the expected result on the reviewed defective candidate, not a passing gate.
- `receipts.json`, `*-inputs.json`: stage commands, execution bounds, toolchain, times, results and source/config hashes, including injected test fixtures. Workstation paths are normalized to `<evidence-root>` in receipts.
- `source-proof.json`, `ci.json`, `pr-snapshot.json`: source equality between #7 and the integrated candidate, existing exact-head full CI and all nine reviewed descriptions.
- `*.go.txt`: review-only fixtures. They are not compiled as part of the repository's suite.
- `artifact-sha256.json`: hashes of this evidence packet, excluding the hash file itself.

Raw pprof files, compiled test binaries, original logs and immutable archive trees remain locally under `.git/review-evidence/final-review-20261108/`. They are omitted from versioned documentation; the inspected text and executable harness are retained here. No production code was modified for the benchmark or defect controls. Compiler/lexer and runtime tests already present in the candidate were not repeatedly run as a new full local suite; the recorded Linux CI supplies that coverage. New edge probes are separate from those passing historical suites.

## Reproduce the defects

Requires the exact commits in the indicated local repositories, Go, Python 3.12+ and the installed `gate-lock`. Use an empty scratch directory. `prepare.py` archives source and writes only inside that directory; it does not modify branches. Set the three repository arguments to checkouts containing the specified commits.

```sh
review_work="$(mktemp -d -t jet-final-review)"
python3 docs/plans/template-execution-budgets/final-review-evidence/prepare.py "$review_work" . ../jet-lane-1 ../jet-lane-3
python3 "$review_work/run_stage.py" run-focused legacy-upstream upstream go test -count=1 -p=1 -parallel=2 -timeout=120s -run TestReviewLegacyPanicBoundary -v .
python3 "$review_work/run_stage.py" run-focused legacy-candidate probes go test -count=1 -p=1 -parallel=2 -timeout=120s -run TestReviewLegacyPanicBoundary -v .
python3 "$review_work/run_stage.py" run-focused final-probes-normal probes go test -count=1 -p=1 -parallel=2 -timeout=120s -run 'TestReview(RestrictedPanic|Sink|Catch)' -v .
python3 "$review_work/run_stage.py" run-focused final-probes-race probes go test -race -count=1 -p=1 -parallel=2 -timeout=120s -run 'TestReview(RestrictedPanic|Sink|Catch)' -v .
```

The legacy controls log the five escaping panics without failing. Restricted probes fail if a panic escapes, or if the expected output/ordinary error differs. The catch and sink probes fail on the documented results. Normal and race expose the same defects; the race instrumentation does not cause them. The extra nil-slice diagnostic probe is available but is not a release finding.

## Reproduce the diagnostic benchmark batch

The recorded harness uses macOS `/usr/bin/time -l`; adapt RSS collection explicitly before using it on another platform. `benchstat` must be on PATH. Source fixtures and options are in the two benchmark text files. The archived corpus contains 15 finite workloads; each runs in text and HTML mode. All three snapshots pass conformance before timed samples.

```sh
REVIEW_TIMEOUT=1200 python3 "$review_work/run_stage.py" run-exclusive benchmarks candidate python3 "$review_work/run_bench.py"
python3 "$review_work/run_stage.py" run-check analysis candidate python3 "$review_work/analyze_bench.py"
```

The first command builds three binaries once, checks conformance, rotates snapshot order across six 100 ms samples, measures setup separately and then captures profiles at 500 ms per selected sub-benchmark. Each child has its own deadline and TERM/KILL cleanup; the outer gate has a 1,200-second execution bound and ten-second cleanup allowance. Queue waiting is not execution time. The second command refuses incomplete/failed benchmark evidence, reads all profile kinds and generates comparisons.

The replay scripts use the same workloads/flags as the recorded run; `analyze_bench.py` resolves benchstat via PATH instead of a workstation-specific path. The release program in the review requires longer samples plus latency, concurrency, overshoot and additional data coverage. This short diagnostic batch does not satisfy those remaining gates.
