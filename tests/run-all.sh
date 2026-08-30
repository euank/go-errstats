#!/usr/bin/env bash

set -euo pipefail

ROOT="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"

cd "$ROOT/tests/case01"

actual="$(mktemp)"
trap 'rm -f "$actual"' EXIT

"$ROOT/errstats" > "$actual"
diff expected.out "$actual"
