# Output finalization artifacts

Read [the report](../finalization-lane-2.md) for scope, statistical limits and pending combined gates. Text copies normalize host paths, remove other owners' queue notices and trim trailing whitespace; raw hashes identify originals in `.git/finalization/`. Profile binaries/pprof remain there.

`selected-measure` preserves six interleaved one-second samples, benchstat intervals, default-sampling CPU/alloc/live/contention/source views and commands. `reserve-*` preserves the short capacity/reuse screens, large-then-small controls and memprofilerate=1 profile views. postGC means GC+render; parallel screens share a context. These are not the final worker-context latency/concurrency matrix.

`variants/0-execution_output.go.txt` plus the zero-context variant patches describe the experimental deltas, not production code. Build/check receipts identify all common inputs, including the injected review benchmark harness. The main executable benchmark harness is retained by the canonical final-review evidence; `capacity-benchmark.go.txt` adds the screen scenarios. `build_capacity.py`, `build_pool.py`, `reserve_screen.py` and `selected_measure.py` record construction/measurement and bounded subprocess cleanup. Run only in a dedicated task checkout under installed host-gate; the construction scripts temporarily change owned files and restore them. Never run concurrent source editors against those files.

The initial historical F3 receipt includes discarded probe mistakes described in the report. Passing selected checks, including final redirect/reentry controls, are separate. No invalid instrumentation or failed baseline is counted as a passing gate.
