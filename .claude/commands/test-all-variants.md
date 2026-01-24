# Test All Variants

Build and test all Docker variants locally on macOS (amd64 architecture).

## Usage

```
/project:test-all-variants
```

## What This Command Does

This command builds and smoke-tests all Docker image variants:
- **Standard** (MySQL, PostgreSQL, SQL Server)
- **MongoDB** (with mongosh)
- **Spanner** (Google Cloud Spanner)

Each variant is built for **amd64** (sufficient for local Mac testing) and tested with the `history` command.

## Steps

### 1. Check Prerequisites

Verify Docker is running:
```bash
docker info > /dev/null 2>&1 || echo "❌ Docker is not running. Please start Docker Desktop."
```

### 2. Build Standard Variant

Build the default Liquibase image:

```bash
echo "Building standard variant..."
docker build -t drone-liquibase:test-standard -f docker/Dockerfile.linux.amd64 . || exit 1
```

### 3. Build MongoDB Variant

Build the MongoDB-specific image:

```bash
echo "Building MongoDB variant..."
docker build -t drone-liquibase:test-mongo -f docker/Dockerfile.linux-mongo.amd64 . || exit 1
```

### 4. Build Spanner Variant

Build the Google Cloud Spanner image:

```bash
echo "Building Spanner variant..."
docker build -t drone-liquibase:test-spanner -f docker/Dockerfile.linux-spanner.amd64 . || exit 1
```

### 5. Smoke Test Each Variant

Test each variant with minimal environment variables to verify it runs:

**Standard:**
```bash
echo "Testing standard variant..."
docker run --rm \
  -e PLUGIN_COMMAND='history' \
  -e PLUGIN_LIQUIBASE_URL='jdbc:h2:mem:testdb' \
  -e PLUGIN_LIQUIBASE_USERNAME='sa' \
  -e PLUGIN_LIQUIBASE_PASSWORD='' \
  drone-liquibase:test-standard || echo "⚠️  Standard variant test failed (expected if no changelog)"
```

**MongoDB:**
```bash
echo "Testing MongoDB variant..."
docker run --rm \
  -e PLUGIN_COMMAND='history' \
  -e PLUGIN_LIQUIBASE_URL='mongodb://localhost:27017/testdb' \
  -e PLUGIN_LIQUIBASE_USERNAME='testuser' \
  -e PLUGIN_LIQUIBASE_PASSWORD='testpass' \
  drone-liquibase:test-mongo || echo "⚠️  MongoDB variant test failed (expected if no connection)"
```

**Spanner:**
```bash
echo "Testing Spanner variant..."
docker run --rm \
  -e PLUGIN_COMMAND='history' \
  -e PLUGIN_LIQUIBASE_URL='jdbc:cloudspanner:/projects/test/instances/test/databases/test' \
  -e PLUGIN_LIQUIBASE_USERNAME='' \
  -e PLUGIN_LIQUIBASE_PASSWORD='' \
  drone-liquibase:test-spanner || echo "⚠️  Spanner variant test failed (expected if no connection)"
```

**Note**: Tests may fail with connection errors if no actual database is available. The goal is to verify:
- ✅ Image builds successfully
- ✅ Entrypoint script runs
- ✅ Required JARs are present
- ✅ No missing dependencies

### 6. Verify JAR Files

For each variant, verify the required JARs are present:

**Standard:**
```bash
docker run --rm drone-liquibase:test-standard ls -la /liquibase/lib/ | grep -E "(dbops-extensions|okhttp|okio|zstd)"
```

**MongoDB:**
```bash
docker run --rm drone-liquibase:test-mongo ls -la /liquibase/lib/ | grep -E "(dbops-extensions|mongodb)"
```

**Spanner:**
```bash
docker run --rm drone-liquibase:test-spanner ls -la /liquibase/lib/ | grep -E "(dbops-extensions|spanner)"
```

### 7. Report Results

Summarize the test results:

```
═══════════════════════════════════════════
    Docker Variant Build & Test Summary
═══════════════════════════════════════════

Standard variant:
  ✓ Build: SUCCESS
  ✓ JAR verification: dbops-extensions-1.17.0.jar, okhttp, okio, zstd found
  ⚠  Smoke test: Failed (expected - no database connection)

MongoDB variant:
  ✓ Build: SUCCESS
  ✓ JAR verification: dbops-extensions-1.17.0.jar, mongodb drivers found
  ⚠  Smoke test: Failed (expected - no database connection)

Spanner variant:
  ✓ Build: SUCCESS
  ✓ JAR verification: dbops-extensions-1.17.0.jar, spanner connector found
  ⚠  Smoke test: Failed (expected - no database connection)

═══════════════════════════════════════════

✅ All variants built successfully!

Note: Smoke test failures are expected without actual database connections.
The builds verified that all dependencies are correctly included.

To clean up test images:
  docker rmi drone-liquibase:test-standard drone-liquibase:test-mongo drone-liquibase:test-spanner
```

## DOs

- Run this command before committing Dockerfile changes
- Verify JAR files are present in each variant
- Check build output for warnings or errors
- Test on macOS (amd64) - CI/CD will handle arm64

## DON'Ts

- Don't skip this command when modifying Dockerfiles
- Don't worry about smoke test connection failures (expected)
- Don't build arm64 variants locally (slow, CI/CD handles it)
- Don't commit if any builds fail

## When to Use This Command

- ✅ After modifying any Dockerfile
- ✅ After upgrading dbops-extensions version
- ✅ After adding/updating download scripts
- ✅ Before creating a pull request with Docker changes

## Cleanup

After testing, clean up test images to save disk space:

```bash
docker rmi drone-liquibase:test-standard drone-liquibase:test-mongo drone-liquibase:test-spanner
```

Or use:
```bash
docker image prune -f
```
