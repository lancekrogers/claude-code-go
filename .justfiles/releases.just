# Release Commands
# Run `just release` to see all release commands

set dotenv-load := true

PROJECT_DIR := justfile_directory()

[private]
@default:
    @echo "Release Commands"
    @echo "================"
    @just --list --justfile {{source_file()}}

# Show current version from latest git tag
version:
    @echo "Current version:"
    @git describe --tags --abbrev=0 2>/dev/null || echo "No tags found"

# Show commits since last release
changelog:
    #!/usr/bin/env bash
    LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null)
    if [ -z "$LAST_TAG" ]; then
        echo "No previous tags found. Showing all commits:"
        git log --oneline
    else
        echo "Changes since $LAST_TAG:"
        git log --oneline "$LAST_TAG"..HEAD
    fi

# Show release readiness status
status:
    #!/usr/bin/env bash
    echo "=== Release Status ==="
    echo ""
    echo "Branch: $(git branch --show-current)"
    echo "Latest tag: $(git describe --tags --abbrev=0 2>/dev/null || echo 'none')"
    echo "Commits since tag: $(git rev-list $(git describe --tags --abbrev=0 2>/dev/null)..HEAD --count 2>/dev/null || echo 'N/A')"
    echo ""
    echo "Uncommitted changes:"
    if git diff --quiet && git diff --staged --quiet; then
        echo "  None (working tree clean)"
    else
        git status --short
    fi

# Run all pre-release checks (lint, test, build)
check:
    @echo "=== Pre-release Checks ==="
    just util lint
    just test all
    just build all
    @echo ""
    @echo "All checks passed!"

# Create signed git tag (use -a instead of -s if GPG not configured)
tag VERSION:
    @echo "Creating signed tag {{VERSION}}..."
    git tag -s {{VERSION}} -m "Release {{VERSION}}"
    @echo "Tag {{VERSION}} created"
    @echo "Run 'just release push' to push to remote"

# Push tags to remote
push:
    @echo "Pushing tags to remote..."
    git push --tags
    @echo "Tags pushed"

# Full release prep: check + tag
prepare VERSION: check
    just release tag {{VERSION}}

# Full release workflow: check + tag + push
publish VERSION: (prepare VERSION)
    just release push
    @echo ""
    @echo "=== Release {{VERSION}} Published ==="

# Generate release notes template
notes VERSION:
    #!/usr/bin/env bash
    LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null)
    echo "# Release {{VERSION}}"
    echo ""
    echo "## What's Changed"
    echo ""
    if [ -z "$LAST_TAG" ]; then
        git log --oneline --pretty=format:"- %s" | head -20
    else
        git log --oneline --pretty=format:"- %s" "$LAST_TAG"..HEAD
    fi
    echo ""
    echo ""
    echo "**Full Changelog**: https://github.com/lancekrogers/claude-code-go/compare/$LAST_TAG...{{VERSION}}"

# Show diff from last release
diff:
    #!/usr/bin/env bash
    LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null)
    if [ -z "$LAST_TAG" ]; then
        echo "No previous tags found"
        exit 1
    fi
    echo "Diff since $LAST_TAG:"
    git diff --stat "$LAST_TAG"..HEAD
