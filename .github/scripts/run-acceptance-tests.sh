#!/usr/bin/env bash
#
# Run the acceptance suite against the local backend, with coverage. Used by
# .github/workflows/test.yml:
#
#   run-acceptance-tests.sh
#
# Required env: the MESHSTACK_* / TF_ACC* variables the step sets (endpoint, credentials, TF_ACC=1).
# Writes covdata/acc (binary coverage-data dir, merged with the unit job's data by
# coverage-comment.sh), junit-acc.xml, and acc-output.log.
set -euo pipefail

mkdir -p covdata/acc
# Declared separately from the export so a missing tofu fails the script here (set -e) rather than
# exporting an empty path and failing obscurely inside the tests.
TF_ACC_TERRAFORM_PATH="$(command -v tofu)"
export TF_ACC_TERRAFORM_PATH

# Select acceptance tests by the `TestAcc` naming convention across ALL packages (./...) rather than
# restricting to ./internal/provider, so acceptance tests added in any package are picked up
# automatically. Coverage is attributed across all packages via -coverpkg=./... so the merged report
# reflects the real end-to-end exercise of the client and helper packages too; it is emitted as a
# binary coverage-data dir for merging with the unit job's data.
#
# Tee the output to a file so the next step can verify the run actually finished: a
# self-hosted-runner truncation can cut the test process short while this step still exits 0, which
# would otherwise pass as a green run. pipefail (set above) keeps a genuine test failure propagating
# through the pipe.
go tool gotestsum --junitfile junit-acc.xml --format testdox -- \
  -coverpkg=./... -count=1 -timeout 10m -run 'TestAcc' ./... \
  -args -test.gocoverdir="$PWD/covdata/acc" 2>&1 | tee acc-output.log
