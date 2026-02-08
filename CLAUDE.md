# SubVault - Claude Code Instructions

## Git Remotes

- **origin** → Lokales Gitea (http://localhost:3001/user/subvault) - push & fetch
- **upstream** → GitHub (https://github.com/bscott/subtrackr) - nur fetch, push blockiert

```bash
# Updates von GitHub holen (nur lesen)
git fetch upstream
git merge upstream/main

# Eigene Änderungen pushen (nur zu Gitea)
git push origin
```

## Release Workflow

This project uses versioned branches for releases. Follow this workflow when working on new features or bug fixes.

### 1. Create a Versioned Branch

```bash
# Check current version
tea releases ls --login local --limit 1

# Create and checkout versioned branch
git checkout -b v0.X.Y
```

### 2. Create Draft Release Before Committing

```bash
# Create draft release with release notes
tea releases create vX.Y.Z --login local --draft \
  --title "vX.Y.Z - Release Title" \
  --note "$(cat <<'EOF'
## What's New

### Feature Name (#issue)
- Description of changes

## Technical Changes
- List of technical changes
EOF
)"
```

### 3. Code Review

Before committing, run the code review agent:
- Check for code quality issues
- Verify security concerns
- Ensure best practices

### 4. Commit and Push

```bash
# Stage changes
git add <files>

# Commit with descriptive message
git commit -m "vX.Y.Z - Release Title

- Change 1
- Change 2"

# Push branch to Gitea
git push -u origin vX.Y.Z
```

### 5. Create Pull Request

```bash
tea pr create --login local \
  --head v0.X.Y \
  --title "vX.Y.Z - Release Title" \
  --description "$(cat <<'EOF'
## Summary
- Change summary

## Test Plan
- [ ] Test item 1
- [ ] Test item 2

Closes #issue1
Closes #issue2
EOF
)"
```

### 6. Comment on Issues

```bash
# Notify issue reporters
tea comment --login local <issue-number> "Fixed in PR #XX. Description of fix."
```

### 7. Merge

```bash
# Merge when ready
tea pr merge <pr-number> --login local --style merge

# Switch to main
git checkout main && git pull
```

### 8. Publish Release

```bash
# Publish the draft release
tea releases edit vX.Y.Z --login local --draft false

# Verify
tea releases ls --login local --limit 1
```

## Git Commit Guidelines

- Do not include AI attribution in commit messages
- Use conventional commit format
- Keep messages clear and descriptive
- Reference GitHub issue numbers where applicable
