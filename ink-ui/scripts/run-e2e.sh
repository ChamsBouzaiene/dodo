#!/bin/bash
# Run E2E tests for Dodo CLI
# Usage: ./run-e2e.sh

set -e

cd "$(dirname "$0")/.."

echo "🧪 Running E2E tests..."
echo ""

# Run vitest with the E2E pattern
# The tests are skipped by default - to enable them, 
# change it.skip() to it() in the test file
npm test -- src/tests/e2e/ --reporter=verbose

echo ""
echo "✅ E2E tests complete!"
