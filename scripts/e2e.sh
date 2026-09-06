#!/usr/bin/env bash
# DevBase end-to-end battery: real binary against synthetic fixtures plus an
# optional live project. Every case asserts its verdict and keeps its evidence.
#
#   bash scripts/e2e.sh                    # fixtures only
#   MEDICOS_DIR=/path/to/proj bash scripts/e2e.sh   # + live record-only case
#
# Evidence lands in .devbase/evidence-e2e/ (gitignored; uploaded in CI).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/devbase-e2e"
EV="$ROOT/.devbase/evidence-e2e"
FIX="$ROOT/testdata/e2e"

go build -o "$BIN" ./cmd/devbase
rm -rf "$EV"
mkdir -p "$EV"

pass=0
fail=0

assert_json() { # file
  python3 - "$1" <<'EOF'
import json, sys
e = json.load(open(sys.argv[1]))
assert e.get("schema") == "devbase.evidence.v1", "bad schema"
assert e.get("verdict") in ("PASS", "FAIL", "INCOMPLETE"), "bad verdict"
assert isinstance(e.get("checks"), list) and e["checks"], "no checks"
EOF
}

check() { # name want_exit dir
  local name="$1" want="$2" dir="$3"
  set +e
  "$BIN" gate --dir "$dir" --json > "$EV/$name.json" 2>"$EV/$name.stderr"
  local got=$?
  assert_json "$EV/$name.json"
  set -e
  if [ "$got" = "$want" ]; then
    echo "PASS $name (exit $got)"
    pass=$((pass + 1))
  else
    echo "FAIL $name: want exit $want, got $got (see $EV/$name.json)"
    fail=$((fail + 1))
  fi
}

check_field() { # name dir jq_expr expected (python)
  local name="$1" dir="$2" expr="$3" want="$4"
  set +e
  "$BIN" gate --dir "$dir" --json > "$EV/$name.json" 2>/dev/null
  local got
  got=$(python3 -c "import json;print($expr)" < "$EV/$name.json")
  set -e
  if [ "$got" = "$want" ]; then
    echo "PASS $name ($expr == $want)"
    pass=$((pass + 1))
  else
    echo "FAIL $name: $expr == $got, want $want"
    fail=$((fail + 1))
  fi
}

echo "== fixtures =="
check clean-go 0 "$FIX/clean-go"
check broken-node 1 "$FIX/broken-node"
check vulnerable-py 1 "$FIX/vulnerable-py"

EMPTY_DIR="$(mktemp -d)"
check empty-dir 2 "$EMPTY_DIR"
rm -rf "$EMPTY_DIR"

echo "== generated dirty tree =="
DIRTY="$(mktemp -d)"
git -C "$DIRTY" init -q
git -C "$DIRTY" -c user.email=t@t -c user.name=t commit -q --allow-empty -m one
echo v2 > "$DIRTY/a.txt"
git -C "$DIRTY" add .
check_field dirty-tree "$DIRTY" "json.load(open('$EV/dirty-tree.json'))['dirty']" "True"
rm -rf "$DIRTY"

if [ -n "${MEDICOS_DIR:-}" ]; then
  echo "== live project (record-only) =="
  set +e
  "$BIN" gate --dir "$MEDICOS_DIR" --json > "$EV/medicos-live.json" 2>"$EV/medicos-live.stderr"
  echo "live verdict: $(python3 -c "import json;print(json.load(open('$EV/medicos-live.json'))['verdict'])") (recorded, not asserted)"
  set -e
fi

rm -f "$BIN"
echo "== $pass passed, $fail failed =="
[ "$fail" = "0" ]
