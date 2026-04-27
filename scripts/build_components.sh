#!/usr/bin/env bash
set -euo pipefail

out_dir="${1:-dist}"
component_dir="$out_dir/components"

mkdir -p "$component_dir"

go build -buildmode=archive -o "$component_dir/repository.a" ./internal/component/repository
go build -buildmode=archive -o "$component_dir/service.a" ./internal/component/service
go build -buildmode=archive -o "$component_dir/techui.a" ./internal/component/techui
go build -o "$out_dir/marketplace-techui" ./cmd/techui
