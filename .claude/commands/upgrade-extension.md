# Upgrade DBOps Extension

Upgrade the dbops-extensions JAR version consistently across all download scripts.

## Usage

```
/project:upgrade-extension <new-version>
```

**Examples:**
- `/project:upgrade-extension 1.18.0` - Upgrade to version 1.18.0
- `/project:upgrade-extension 1.17.1` - Patch version upgrade

## What This Command Does

This command ensures version consistency by:

1. **Identifying current version** across all scripts
2. **Updating version** in all download scripts
3. **Verifying consistency** after update
4. **Documenting the change** for commit message

## Steps

### 1. Detect Current Version

Read the current version from:
- `scripts/download-common.sh` - Line with `LIQUIBASE_DBOPS_EXT_VERSION=`

```bash
grep "LIQUIBASE_DBOPS_EXT_VERSION=" scripts/download-common.sh
```

Report to user:
```
Current dbops-extensions version: 1.17.0
Upgrading to: {new-version}
```

### 2. Validate New Version

Check if the new version exists in the Harness repository (if accessible):
```bash
# This is informational - proceed even if check fails
curl -I "https://harness0.harness.io/.../dbops-extensions-{version}.jar" 2>/dev/null || echo "Note: Unable to verify JAR existence (may require auth)"
```

Ask user to confirm:
```
⚠️  Make sure the dbops-extensions-{version}.jar file exists in the Harness repository before proceeding.

Proceed with upgrade? (Y/n)
```

### 3. Update Download Scripts

Update `LIQUIBASE_DBOPS_EXT_VERSION` in:

**Primary file:**
- `scripts/download-common.sh`

**Pattern:**
```bash
LIQUIBASE_DBOPS_EXT_VERSION="1.17.0"
```

**Replace with:**
```bash
LIQUIBASE_DBOPS_EXT_VERSION="{new-version}"
```

**Other files to check** (may reference the version):
- `AGENTS.md` - Update "Extension Version" section if present
- `README.md` - Update version references if present

### 4. Verify Consistency

After update, verify all files reference the new version:

```bash
# Should return only the new version
grep -r "LIQUIBASE_DBOPS_EXT_VERSION" scripts/ | grep -v "^#"
grep -r "dbops-extensions-" . --include="*.md" --include="*.sh" | grep -E "[0-9]+\.[0-9]+\.[0-9]+"
```

Report any inconsistencies to the user.

### 5. Update Documentation

Update `AGENTS.md`:

Find and replace in the "Key Components" or "Extension Version" section:
```markdown
- **Extension**: DBOps {old-version}
```

With:
```markdown
- **Extension**: DBOps {new-version}
```

### 6. Generate Commit Message

Suggest commit message following project format:

```
[feat]: [DBOPS-XXXX]: Upgrade dbops-extensions to {new-version}

- Updated LIQUIBASE_DBOPS_EXT_VERSION in scripts/download-common.sh
- Updated documentation references in AGENTS.md
- Version: {old-version} → {new-version}
```

Ask user for Jira ticket number to complete the message.

### 7. Testing Recommendations

Inform the user:

```
🧪 Recommended testing steps:

1. Build default image for macOS (amd64):
   docker build -f docker/Dockerfile.linux.amd64 .

2. Test with the history command:
   docker run --rm \
     -e PLUGIN_COMMAND='history' \
     -e PLUGIN_LIQUIBASE_URL='jdbc:postgresql://localhost:5432/testdb' \
     -e PLUGIN_LIQUIBASE_USERNAME='testuser' \
     -e PLUGIN_LIQUIBASE_PASSWORD='testpass' \
     <image-id>

Proceed with commit after successful testing.
```

## DOs

- Always verify version format is semantic versioning (X.Y.Z)
- Update all references to the version, not just the main variable
- Test default image build (amd64) before committing
- Document the upgrade in commit message
- Check if AGENTS.md references the version

## DON'Ts

- Don't skip version validation
- Don't commit without testing the build
- Don't forget to update documentation
- Don't use non-semantic version formats

## Example Output

```
Current version: 1.17.0
New version: 1.18.0

Updated files:
  ✓ scripts/download-common.sh (line 4: LIQUIBASE_DBOPS_EXT_VERSION="1.18.0")
  ✓ AGENTS.md (updated Extension version reference)

Verification:
  ✓ All version references consistent (1.18.0)
  ✓ No stale version strings found

Suggested commit message:
[feat]: [DBOPS-XXXX]: Upgrade dbops-extensions to 1.18.0

- Updated LIQUIBASE_DBOPS_EXT_VERSION in scripts/download-common.sh
- Updated documentation references in AGENTS.md
- Version: 1.17.0 → 1.18.0

Next steps:
1. Enter Jira ticket number: DBOPS-____
2. Test build: docker build -f docker/Dockerfile.linux.amd64 .
3. Test with: PLUGIN_COMMAND=history, PLUGIN_LIQUIBASE_URL, USERNAME, PASSWORD
4. Commit: git add scripts/download-common.sh AGENTS.md
5. Use suggested commit message
```

## Version History Tracking

After successful upgrade, this command helps maintain a clear audit trail:
- Version changes are explicit in commit messages
- Documentation stays synchronized
- All scripts reference the same version
