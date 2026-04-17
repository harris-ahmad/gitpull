#!/bin/bash
set -e

GITPULL="$(pwd)/gitpull"
TEST_DIR="/tmp/gitpull-test-$$"
REMOTE_DIR="$TEST_DIR/remotes"
WORKSPACE="$TEST_DIR/workspace"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

pass() {
    echo -e "${GREEN}✓ PASS${NC} $1"
}

fail() {
    echo -e "${RED}✗ FAIL${NC} $1"
    exit 1
}

# Build gitpull if not exists
if [ ! -f "$GITPULL" ]; then
    log "Building gitpull..."
    go build -o gitpull .
fi

# Cleanup previous test run
rm -rf "$TEST_DIR"
mkdir -p "$REMOTE_DIR" "$WORKSPACE"

log "Creating test workspace in $TEST_DIR"

# ============================================================================
# Setup: Create remote repos
# ============================================================================

setup_repo() {
    local name=$1
    local remote="$REMOTE_DIR/$name"
    local local="$WORKSPACE/$name"

    # Create initial repo with a commit
    local tmp="/tmp/gitpull-setup-$$-$name"
    mkdir -p "$tmp"
    cd "$tmp"
    git init -q
    git config user.email "test@test.com"
    git config user.name "Test User"
    echo "# $name" > README.md
    git add README.md
    git commit -q -m "initial commit"

    # Create bare remote and push
    git init -q --bare "$remote"
    git remote add origin "$remote"
    git push -q origin main

    # Clone to workspace
    git clone -q "$remote" "$local"
    cd "$local"
    git config user.email "test@test.com"
    git config user.name "Test User"

    # Cleanup temp
    rm -rf "$tmp"
}

log "Setting up test repositories..."

# Repo 1: Clean repo, behind remote
setup_repo "clean-behind"
cd "$WORKSPACE/clean-behind"
# Make a commit in the remote
tmp="/tmp/gitpull-remote-$$"
git clone -q "$REMOTE_DIR/clean-behind" "$tmp"
cd "$tmp"
git config user.email "test@test.com"
git config user.name "Test User"
echo "new feature" >> README.md
git add README.md
git commit -q -m "feat: add new feature"
git push -q origin main
rm -rf "$tmp"

# Repo 2: Dirty repo, behind remote
setup_repo "dirty-behind"
cd "$WORKSPACE/dirty-behind"
# Make local changes
echo "work in progress" >> local-work.txt
git add local-work.txt
# Make remote ahead
tmp="/tmp/gitpull-remote-$$"
git clone -q "$REMOTE_DIR/dirty-behind" "$tmp"
cd "$tmp"
git config user.email "test@test.com"
git config user.name "Test User"
echo "remote change" >> README.md
git add README.md
git commit -q -m "feat: remote work"
git push -q origin main
rm -rf "$tmp"

# Repo 3: Repo with merged branch (and behind remote so cleanup runs)
setup_repo "merged-branch"
cd "$WORKSPACE/merged-branch"
git checkout -q -b feature/test
echo "feature work" >> feature.txt
git add feature.txt
git commit -q -m "feat: feature work"
git checkout -q main
git merge -q --no-ff feature/test -m "merge feature"
# Now feature/test is merged but branch still exists locally

# Make remote ahead so sync will pull (and trigger cleanup)
tmp="/tmp/gitpull-remote-$$"
git clone -q "$REMOTE_DIR/merged-branch" "$tmp"
cd "$tmp"
git config user.email "test@test.com"
git config user.name "Test User"
echo "more work" >> README.md
git add README.md
git commit -q -m "feat: more work"
git push -q origin main
rm -rf "$tmp"

# Repo 4: Up to date repo
setup_repo "up-to-date"

# Repo 5: Conflict scenario
setup_repo "conflict"
cd "$WORKSPACE/conflict"
echo "local version" > conflict.txt
git add conflict.txt
# Make conflicting remote change
tmp="/tmp/gitpull-remote-$$"
git clone -q "$REMOTE_DIR/conflict" "$tmp"
cd "$tmp"
git config user.email "test@test.com"
git config user.name "Test User"
echo "remote version" > conflict.txt
git add conflict.txt
git commit -q -m "feat: remote change"
git push -q origin main
rm -rf "$tmp"

# ============================================================================
# Test 1: Basic sync (clean-behind should pull) AND auto-stash/auto-pop AND branch cleanup
# ============================================================================

log "Test 1: Running comprehensive sync test with --cleanup"
cd "$WORKSPACE"
$GITPULL sync --cleanup 2>&1 | tee /tmp/gitpull-test-output.txt

cd "$WORKSPACE/clean-behind"
if git log --oneline | grep -q "feat: add new feature"; then
    pass "Test 1a: Clean repo pulled successfully"
else
    fail "Test 1a: Failed to pull clean repo"
fi

cd "$WORKSPACE/dirty-behind"
if [ -f "local-work.txt" ]; then
    pass "Test 1b: Local work restored after auto-stash/pull"
else
    fail "Test 1b: Local work lost during auto-stash!"
fi

if git log --oneline | grep -q "feat: remote work"; then
    pass "Test 1c: Remote changes pulled despite local changes"
else
    fail "Test 1c: Failed to pull remote changes"
fi

cd "$WORKSPACE/merged-branch"
if git branch | grep -q "feature/test"; then
    fail "Test 1d: Merged branch was not deleted with --cleanup"
else
    pass "Test 1d: Merged branch cleaned up successfully"
fi

# ============================================================================
# Test 2: Up to date repo
# ============================================================================

log "Test 2: Up to date repo should report correctly"
cd "$WORKSPACE"
output=$($GITPULL sync 2>&1)
if echo "$output" | grep -q "up-to-date.*up to date"; then
    pass "Test 2: Up to date repo reported correctly"
else
    fail "Test 2: Up to date repo not reported"
fi

# ============================================================================
# Test 3: Conflict detection
# ============================================================================

log "Test 3: Conflict should be detected and repo skipped"
cd "$WORKSPACE"
output=$($GITPULL sync 2>&1)
if echo "$output" | grep -q "conflict.*conflict predicted"; then
    pass "Test 3a: Conflict detected"
else
    fail "Test 3a: Conflict not detected"
fi

cd "$WORKSPACE/conflict"
if [ -f "conflict.txt" ] && grep -q "local version" conflict.txt; then
    pass "Test 3b: Local changes preserved when conflict detected"
else
    fail "Test 3b: Local changes lost or modified"
fi

# ============================================================================
# Cleanup
# ============================================================================

log "Cleaning up test workspace..."
rm -rf "$TEST_DIR"

echo ""
echo -e "${GREEN}═══════════════════════════════════${NC}"
echo -e "${GREEN}   ALL TESTS PASSED ✓${NC}"
echo -e "${GREEN}═══════════════════════════════════${NC}"
echo ""
