#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "🧪 Running backend test suite with race detector and coverage..."
test_output=$(go test -v -race -cover ./internal/...)
echo "$test_output"

echo "📊 Verifying 100% statement coverage across all packages..."
echo "$test_output" | awk '
/^ok[ \t]+/ && /coverage:/ {
  count++
  for (i=1; i<=NF; i++) {
    if ($i == "coverage:") {
      pct = $(i+1)
      if (pct != "100.0%") {
        print "❌ Coverage failure: " $0
        fail = 1
      }
    }
  }
}
END {
  if (count < 16) {
    print "❌ Expected 16 packages, only verified " count
    exit 1
  }
  if (fail) {
    print "❌ Not all backend packages reached 100.0% statement coverage!"
    exit 1
  }
  print "✅ All " count " backend packages reached 100.0% statement coverage!"
}'
