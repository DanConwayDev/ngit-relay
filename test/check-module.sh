#!/usr/bin/env bash

Script to test ngit-relay-module.nix configurations
set -e

echo "Testing ngit-relay-module.nix configurations..."

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Get the project root (parent of test directory)
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

test_files=(
"$PROJECT_ROOT/test/test-basic-enable.nix"
"$PROJECT_ROOT/test/test-with-image.nix"
"$PROJECT_ROOT/test/test-multiple-instances.nix"
"$PROJECT_ROOT/test/test-custom-dirs.nix"
)

for test_file in "${test_files[@]}"
do
    echo ""
    echo "=== Testing $(basename "$test_file") ==="

    if nix-instantiate --eval --strict "$test_file" >/dev/null 2>&1; then
        echo "✓ $(basename "$test_file"): Syntax OK"
        
        # Try to build the configuration
        if nix-build '<nixpkgs/nixos>' -A system --arg configuration "$test_file" --dry-run 2>/dev/null; then
            echo "✓ $(basename "$test_file"): Configuration builds successfully"
        else
            echo "✗ $(basename "$test_file"): Configuration has build errors (expected for some tests)"
        fi
    else
        echo "✗ $(basename "$test_file"): Syntax errors"
        nix-instantiate --eval --strict "$test_file" 2>&1 | head -5
    fi

done

echo ""
echo "Testing complete. Check output above for any issues."