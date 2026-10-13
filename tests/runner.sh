#!/usr/bin/env bash
set -euo pipefail
TEST_WORKSPACE="${TEST_WORKSPACE:-/workspace}"
if [[ -n "${TEST_REPOSITORY_URL:-}" ]]; then
    git init -q "$TEST_WORKSPACE"
    if git -C "$TEST_WORKSPACE" remote get-url origin >/dev/null 2>&1; then
        git -C "$TEST_WORKSPACE" remote set-url origin "$TEST_REPOSITORY_URL"
    else
        git -C "$TEST_WORKSPACE" remote add origin "$TEST_REPOSITORY_URL"
    fi
    git -C "$TEST_WORKSPACE" fetch --depth=1 origin "$TEST_COMMIT_SHA"
    git -C "$TEST_WORKSPACE" checkout -q --detach FETCH_HEAD
else
    tar xf /run/source.tar -C "$TEST_WORKSPACE"
fi
cd "$TEST_WORKSPACE"
exec python3 -m scripts.testing.runner "$@"
