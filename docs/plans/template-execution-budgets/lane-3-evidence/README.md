# Lane 3 validation evidence

Stage commands, deadlines, exact source inputs and results are in [stage-receipts.json](stage-receipts.json). Local manifests include dirty/untracked source and dependency/toolchain configuration. CI links identify immutable tested commits; normal and race jobs are distinct.

Published logs omit queue messages about unrelated workloads and replace local workspace/home paths with `$CHECKOUT`/`$HOME`. Source-file hashes are unchanged. Raw logs remain under `.git/lane3-evidence` in the task checkout. The archived baseline Go test is text, runs crash probes only in externally bounded children, and is not a normal test-package input.

The cleanup baseline has two independent outcomes: its template test failed as intended, and gate release returned 125. The exact owned admission was recovered through installed gate-lock after process exit. The initial historical PID match was rejected before mutation; recovery then matched the current nonce/supervisor. No lock files were removed manually.

[Signature results](signature-verification.json) verify all new canonical/review commits locally and on GitHub. The consumed original cycle commit remains unsigned with its original author. [Fork review refs](fork-review-refs.json) record exact dependent PR bases. No new upstream PR is authorized.
