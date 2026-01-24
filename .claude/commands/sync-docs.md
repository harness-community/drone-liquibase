# Sync Documentation

Keep AGENTS.md and README.md synchronized with code changes.

## Usage

```
/project:sync-docs
```

## What This Command Does

This command ensures documentation stays in sync with code by:

1. **Detecting code changes** since last commit
2. **Identifying documentation gaps** in AGENTS.md and README.md
3. **Suggesting updates** to keep docs current
4. **Optionally updating** documentation based on detected changes

## Steps

### 1. Detect Recent Changes

Check what files have been modified:

```bash
# Check staged changes
git diff --cached --name-only

# Check unstaged changes
git diff --name-only

# Check recent commits (last 3)
git log -3 --oneline --name-only
```

Categorize changes:
- **Dockerfiles**: `docker/Dockerfile.*`
- **Scripts**: `scripts/*.sh`
- **Entrypoint**: `entrypoint.sh`
- **Resources**: `resources/global_options.txt`

### 2. Analyze Documentation State

**Read current documentation:**
- `AGENTS.md` - Full technical documentation
- `README.md` - User-facing documentation

**Extract key information:**
- Docker Images table (AGENTS.md line ~85)
- Supported databases list (README.md)
- Environment variables table (AGENTS.md line ~145)
- Version numbers (both files)

### 3. Detect Gaps

Based on code changes, check for documentation gaps:

#### New Database Variant Added

**Trigger**: New `docker/Dockerfile.linux-{db}.amd64` file exists

**Required updates:**

AGENTS.md:
- Add row to Docker Images table (line ~85):
  ```markdown
  | {Database} | `liquibase/liquibase:4.33-alpine` | {Database} with JDBC driver |
  ```
- Add to "Building Docker Images Locally" section (line ~95):
  ```bash
  # {Database} image (amd64)
  docker build -t drone-liquibase-{db} -f docker/Dockerfile.linux-{db}.amd64 .
  ```

README.md:
- Add to supported databases list

#### Version Upgrade

**Trigger**: `LIQUIBASE_DBOPS_EXT_VERSION` changed in `scripts/download-common.sh`

**Required updates:**

AGENTS.md:
- Update "Extension Version" line if present

#### New Global Option Added

**Trigger**: New line in `resources/global_options.txt`

**Required updates:**

AGENTS.md:
- Add example in "Adding a New Liquibase Option" section (line ~195)

#### Entrypoint Logic Changed

**Trigger**: `entrypoint.sh` modified

**Required updates:**

AGENTS.md:
- Review "Key Components > entrypoint.sh" section (line ~47)
- Update function table if new functions added
- Update line number references if structure changed

### 4. Present Suggested Updates

Show user what needs updating:

```
Documentation Sync Check
═══════════════════════════════════════════

Detected changes:
  • New Dockerfile: docker/Dockerfile.linux-oracle.amd64
  • Version bump: dbops-extensions 1.17.0 → 1.18.0
  • Modified: entrypoint.sh (SSL/TLS section)

Suggested documentation updates:

📄 AGENTS.md:
  [ ] Add Oracle to Docker Images table (line 85)
  [ ] Add Oracle build command (line 95)
  [ ] Update extension version reference (line 15)
  [ ] Review entrypoint.sh line numbers in Key Components section

📄 README.md:
  [ ] Add Oracle to supported databases list

Would you like me to apply these updates? (Y/n)
```

### 5. Apply Updates (if confirmed)

Update AGENTS.md and README.md based on detected changes.

**For new database variant:**

1. Read existing Docker Images table
2. Insert new row alphabetically
3. Add build command to appropriate section
4. Update README.md databases list

**For version changes:**

1. Find all version references
2. Update to new version
3. Verify consistency across both files

**For entrypoint changes:**

1. Ask user to describe the change
2. Suggest which section to update
3. Let user review before applying

### 6. Verify Consistency

After updates, verify:

```bash
# Check version consistency
grep -E "[0-9]+\.[0-9]+\.[0-9]+" AGENTS.md README.md | sort -u

# Check database list matches Dockerfiles
ls docker/Dockerfile.linux-*.amd64 | sed 's/.*linux-//' | sed 's/\.amd64//' | sort > /tmp/dockerfiles.txt
grep -A 20 "Database Support" README.md | grep "^-" | sed 's/^- //' | tr '[:upper:]' '[:lower:]' | sort > /tmp/readme-dbs.txt
diff /tmp/dockerfiles.txt /tmp/readme-dbs.txt || echo "⚠️  Database lists don't match"
```

### 7. Report Results

```
Documentation sync complete!

Updated files:
  ✓ AGENTS.md (3 changes)
    - Added Oracle to Docker Images table
    - Updated extension version to 1.18.0
    - Added Oracle build command example

  ✓ README.md (1 change)
    - Added Oracle to supported databases

Verification:
  ✓ Version references consistent across both files
  ✓ Database lists match Dockerfiles
  ✓ No broken internal links

Next steps:
  1. Review changes: git diff AGENTS.md README.md
  2. Commit with code changes: git add AGENTS.md README.md <other-files>
  3. Use descriptive commit message mentioning documentation updates
```

## DOs

- Run this command before committing code changes
- Keep version numbers synchronized across AGENTS.md and README.md
- Update both user-facing (README) and technical (AGENTS.md) docs
- Verify database lists match actual Dockerfiles

## DON'Ts

- Don't commit code changes without syncing documentation
- Don't update only one of the two documentation files
- Don't forget to update line number references in AGENTS.md when structure changes
- Don't make documentation changes without reviewing the diff

## When to Use This Command

- ✅ After adding a new database variant
- ✅ After upgrading dbops-extensions version
- ✅ After modifying entrypoint.sh logic
- ✅ After adding new Liquibase global options
- ✅ Before creating a pull request
- ✅ During code review if docs are out of sync

## Example Workflow

```bash
# 1. Make code changes
vim docker/Dockerfile.linux-oracle.amd64
vim scripts/download-oracle.sh

# 2. Before committing, sync docs
/project:sync-docs

# 3. Review changes
git diff AGENTS.md README.md

# 4. Commit everything together
git add docker/ scripts/ AGENTS.md README.md
git commit -m "[feat]: [DBOPS-1234]: Add Oracle database support with documentation"
```
