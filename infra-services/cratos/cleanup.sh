#!/bin/bash

# Cleanup script to remove obsolete files and prevent VS Code caching issues
# Run this script whenever old files reappear

set -e

echo "🧹 Starting comprehensive cleanup..."

# Navigate to project root
cd "$(dirname "$0")"

# 1. Remove obsolete directories and files

# 2. Clean build artifacts
echo "Cleaning build artifacts..."
find . -name "bin" -type d -exec rm -rf {} + 2>/dev/null || true
find . -name "*.test" -delete 2>/dev/null || true
find . -name "*.out" -delete 2>/dev/null || true
find . -name "coverage.html" -delete 2>/dev/null || true

# 3. Clean Go module cache
echo "Cleaning Go module cache..."
cd src && go clean -modcache 2>/dev/null || true
cd ..

# 4. Clean message bus data
echo "Cleaning message bus data..."
rm -rf src/component 2>/dev/null || true
rm -rf /tmp/cratos-messagebus/ 2>/dev/null || true

# 5. Clean test results
echo "Cleaning test results..."
rm -rf test/results/ 2>/dev/null || true
rm -rf test/coverage 2>/dev/null || true
rm -rf test/service.pid 2>/dev/null || true

# 6. Clean individual module artifacts
echo "Cleaning module artifacts..."
if [ -d "src/service" ]; then
    cd src/service && make clean 2>/dev/null || true
    cd ../..
fi

if [ -d "src/testrunner" ]; then
    cd src/testrunner && make clean 2>/dev/null || true
    cd ../..
fi

if [ -d "src/shared" ]; then
    cd src/shared && make clean 2>/dev/null || true
    cd ../..
fi

# 7. Git cleanup

# 8. Remove empty files (except those being worked on)
echo "Removing empty files..."
find . -type f -empty -not -path "./.git/*" -not -name "coverage_improvement_test.go" -delete 2>/dev/null || true

# 9. Clear VS Code workspace state (optional - will reset your open files)
# Uncomment the next line if you want to reset VS Code completely
# rm -rf .vscode/settings.json.bak .vscode/*.log 2>/dev/null || true

echo "✅ Cleanup completed!"
echo ""
echo "🔄 To prevent files from reappearing:"
echo "1. Reload VS Code window: Cmd+Shift+P -> 'Developer: Reload Window'"
echo "2. If issues persist, restart VS Code completely"
echo "3. Run this script again if old files reappear"
echo ""
echo "📁 Current clean state:"
ls -la src/ | grep -E "(service|testrunner|shared|README\.md|Makefile|go\.work)" | echo "Basic files present"
