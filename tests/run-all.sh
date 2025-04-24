#!/usr/bin/env bash

set -eux

ROOT="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"

cd $ROOT/tests/case01

$ROOT/errstats > actual.out
diff expected.out actual.out
