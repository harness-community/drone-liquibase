# Add Database Variant

Create a new database variant for the Liquibase Drone plugin with multi-architecture support.

## Usage

```
/project:add-db-variant <database-name>
```

**Examples:**
- `/project:add-db-variant oracle` - Add Oracle database support
- `/project:add-db-variant cassandra` - Add Cassandra database support

**Reference Implementation:**
See the Snowflake variant for a complete example that follows the standard pattern with Snowflake JDBC driver preserved.

## What This Command Does

This command scaffolds a complete new database variant by creating:

1. **Dockerfiles** (both architectures):
   - `docker/Dockerfile.linux-{db}.amd64`
   - `docker/Dockerfile.linux-{db}.arm64`

2. **Download script**:
   - `scripts/download-{db}.sh` (fetches database-specific JARs)

3. **Manifest template**:
   - `docker/manifest-{db}.tmpl` (multi-arch image manifest)

4. **Documentation updates**:
   - Updates `AGENTS.md` with new variant info
   - Updates `README.md` with new database type

## Steps

### 1. Gather Information

Ask the user:
- **Database name** (if not provided in command)
- **JDBC driver details**: Maven coordinates or download URL
- **Additional dependencies**: Any extra JARs needed
- **Base image preference**: Alpine or UBI9 (Alpine default, UBI9 for special cases like MongoDB)

### 2. Create Download Script

Create `scripts/download-{db}.sh` for downloading database-specific JDBC drivers and dependencies.

**Reference files:**
- `scripts/download-mongo.sh` - MongoDB example
- `scripts/download-spanner.sh` - Spanner example

### 3. Create Dockerfiles

Create `docker/Dockerfile.linux-{db}.amd64` and `docker/Dockerfile.linux-{db}.arm64` for both architectures.

**Reference files:**
- `docker/Dockerfile.linux-snowflake.amd64` - Alpine-based standard pattern (best starting point)
- `docker/Dockerfile.linux-mongo.amd64` - UBI9-based pattern (for compatibility issues)
- `docker/Dockerfile.linux-spanner.amd64` - Debian-based with custom downloader stage

**Key points:**
- Use multi-stage build pattern
- **ALWAYS remove Snowflake JDBC driver** (unless creating Snowflake variant): `rm -f /liquibase/internal/lib/snowflake-jdbc.jar`
- Remove commercial JARs: `liquibase-checks.jar`, `liquibase-commercial-bigquery.jar`
- Prefer Alpine base; use UBI9/Debian only for compatibility needs
- ARM64 Dockerfile is identical to AMD64 (just different filename)

### 4. Create Manifest Template

Create `docker/manifest-{db}.tmpl` for multi-architecture image manifest.

**Reference files:**
- `docker/manifest-snowflake.tmpl`
- `docker/manifest-mongo.tmpl`
- `docker/manifest-spanner.tmpl`

**CRITICAL**: The variant suffix (`-{db}`) must come **AFTER** the version/latest, not before.
- ✓ Correct: `plugins/drone-liquibase:1.18.0-4.33-mongo`
- ✗ Wrong: `plugins/drone-liquibase:mongo-1.18.0-4.33`

Copy any reference manifest and replace the database name suffix appropriately.

### 5. Update Documentation

#### Update AGENTS.md

Add new row to the "Docker Images" table (around line 85) following the existing pattern.

Add build command to "Building Docker Images Locally" section following the existing examples.

#### Update README.md

Add new database to the "Supported Database Types" section with:
- Description following the pattern: `**{Database}** ('*-{db}'): {Description} (Snowflake JDBC driver excluded)`
- Usage example following the existing format with `PLUGIN_IMAGE_{DB}=` variables

**Note**: Use "(Snowflake JDBC driver included)" only for Snowflake variant; all others should say "excluded".

### 6. Verification Steps

After creation, inform the user to:

1. **Test the download script**:
   ```bash
   docker build -t test-{db} -f docker/Dockerfile.linux-{db}.amd64 --target liquibase-builder .
   docker run --rm test-{db} ls -la /liquibase/lib/
   ```

2. **Build both architectures**:
   ```bash
   docker build -t drone-liquibase-{db}:amd64 -f docker/Dockerfile.linux-{db}.amd64 .
   docker buildx build --platform linux/arm64 -t drone-liquibase-{db}:arm64 -f docker/Dockerfile.linux-{db}.arm64 .
   ```

3. **Test the variant**:
   ```bash
   docker run --rm \
     -e PLUGIN_COMMAND='--version' \
     drone-liquibase-{db}:amd64
   ```

4. **Add to CI/CD pipeline** (manual step - document where)

## DOs

- Follow existing naming convention: `{db}` should be lowercase, hyphen-separated
- Always create both amd64 and arm64 Dockerfiles
- Include download script even if no extra JARs (can be minimal)
- Test builds before committing
- Update both AGENTS.md and README.md

## DON'Ts

- Don't skip the manifest template (needed for multi-arch support)
- Don't hardcode versions in Dockerfiles (use build args if needed)
- Don't forget to make download scripts executable (`chmod +x`)
- Don't change the base Liquibase version without team discussion

## Example Output

```
Created files:
  ✓ docker/Dockerfile.linux-oracle.amd64
  ✓ docker/Dockerfile.linux-oracle.arm64
  ✓ scripts/download-oracle.sh
  ✓ docker/manifest-oracle.tmpl

Updated files:
  ✓ AGENTS.md (added Oracle to Docker Images table)
  ✓ README.md (added Oracle to supported databases)

Next steps:
1. Test download script: docker build -f docker/Dockerfile.linux-oracle.amd64 --target liquibase-builder .
2. Build both architectures
3. Test connection to Oracle database
4. Add to CI/CD pipeline
5. Commit: git add -A && git commit -m "[feat]: [DBOPS-XXXX]: Add Oracle database variant support"
```
