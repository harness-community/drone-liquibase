# AGENTS.md

This file provides context and instructions for AI coding agents working on the Harness Drone Liquibase Plugin.

## Project Overview

This is a **Drone CI/CD plugin** that wraps the official [Liquibase Docker image](https://github.com/liquibase/docker) to provide database schema version control and change management capabilities in CI/CD pipelines.

The plugin acts as an **adapter layer** that:
- Translates Drone plugin environment variables (`PLUGIN_*`) into Liquibase CLI arguments
- Handles SSL/TLS certificate configuration for secure database connections
- Integrates with the Harness DBOps platform via the `dbops-extensions` JAR
- Supports multiple database types: MySQL, PostgreSQL, MongoDB, Google Cloud Spanner, DynamoDB, Databricks

**Related Repository**: The `dbops-extensions` JAR that this plugin uses is maintained at `https://harness0.harness.io/ng/account/l7B_kbSEQD2wjrM7PShm5w/module/code/orgs/default/projects/CD/repos/dbopsExtensions` - see its [AGENTS.md](../dbopsExtensions/AGENTS.md) for extension-specific details.

## Repository Structure

```
drone-liquibase/
├── entrypoint.sh              # Main orchestration script (Docker entrypoint)
├── resources/
│   └── global_options.txt     # Whitelist of 144 Liquibase global options
├── docker/
│   ├── Dockerfile.linux.amd64         # Standard image (Alpine-based)
│   ├── Dockerfile.linux.arm64         # ARM64 variant
│   ├── Dockerfile.linux-mongo.amd64   # MongoDB image (UBI9-based)
│   ├── Dockerfile.linux-mongo.arm64   # MongoDB ARM64 variant
│   ├── Dockerfile.linux-spanner.amd64 # Spanner image
│   ├── Dockerfile.linux-spanner.arm64 # Spanner ARM64 variant
│   ├── local                          # Local development Dockerfile
│   ├── manifest.tmpl                  # Multi-arch manifest template
│   ├── manifest-mongo.tmpl            # MongoDB manifest template
│   └── manifest-spanner.tmpl          # Spanner manifest template
├── scripts/
│   ├── download-common.sh     # Downloads common dependency JARs
│   ├── download-mongo.sh      # Downloads MongoDB-specific JARs
│   ├── download-spanner.sh    # Downloads Spanner connector
│   ├── install-mongosh.sh     # Installs MongoDB shell
│   └── publish-docker.sh      # Local publish script for testing
├── dbops-extensions-1.0.0.jar # DBOps extension JAR (copied during build)
├── README.md                  # User-facing documentation
├── LICENSE                    # Apache 2.0 license
└── NOTICE                     # Third-party notices
```

## Key Components

### 1. entrypoint.sh (Lines 1-285)

The main bash script that orchestrates the plugin execution. Understanding this file is essential.

**Key Functions:**

| Section | Lines | Purpose |
|---------|-------|---------|
| Dependency Check | 4-9 | Validates `jq`, `zstd`, `base64` are available |
| `read_global_options()` | 12-52 | Reads options with retry mechanism |
| Java Home Detection | 54-60 | Locates Java installation |
| SSL/TLS TrustStore Setup | 80-112 | Imports root CA certificate |
| SSL/TLS KeyStore Setup | 114-149 | Imports client certificate for mTLS |
| Global Options Processing | 177-187 | Converts `PLUGIN_LIQUIBASE_*` to CLI args |
| Substitution Properties | 192-227 | Decodes base64+zstd JSON to `-D` args |
| Remaining Env Vars | 230-238 | Processes any leftover `PLUGIN_LIQUIBASE_*` vars |
| Google Cloud Auth | 241-251 | Writes service account key file |
| Command Execution | 268-271 | Runs Liquibase with constructed arguments |
| Output Handling | 273-285 | Writes exit code and step outputs |

**Environment Variable Translation Pattern:**
```bash
PLUGIN_LIQUIBASE_LOG_LEVEL=DEBUG  →  --log-level DEBUG
PLUGIN_LIQUIBASE_SEARCH_PATH=/changelog  →  --search-path /changelog
```

### 2. resources/global_options.txt

A whitelist of Liquibase global options (144 entries). Each line is an option name that can be set via environment variable.

**Adding a new option:**
1. Add the option name (kebab-case) to this file
2. Set via `PLUGIN_LIQUIBASE_<OPTION_NAME>` (uppercase, underscores)

### 3. Docker Images

Three image variants are built:

| Variant | Base Image | Use Case |
|---------|------------|----------|
| Standard | `liquibase/liquibase:4.33-alpine` | MySQL, PostgreSQL, SQL Server |
| MongoDB | `liquibase/liquibase:4.33` (UBI9) | MongoDB with mongosh |
| Spanner | `liquibase/liquibase:4.33-alpine` | Google Cloud Spanner |

## Setup Commands

### Building Docker Images Locally

```bash
# Standard image (amd64)
docker build -t drone-liquibase -f docker/Dockerfile.linux.amd64 .

# MongoDB image (amd64)
docker build -t drone-liquibase-mongo -f docker/Dockerfile.linux-mongo.amd64 .

# Spanner image (amd64)
docker build -t drone-liquibase-spanner -f docker/Dockerfile.linux-spanner.amd64 .

# Cross-platform build on ARM Mac
docker buildx build --platform linux/amd64 --load -t drone-liquibase -f docker/Dockerfile.linux.amd64 .
```

### Local Development Dockerfile

For local testing with custom dbops-extensions JAR:

```bash
# Copy the extension JAR to project root
cp /path/to/dbops-extensions-1.0.0.jar .

# Build using local Dockerfile
docker build -t drone-liquibase:local -f docker/local .
```

### Testing the Plugin

```bash
docker run --rm \
  -v /path/to/changelog:/liquibase/changelog \
  -e PLUGIN_COMMAND='update' \
  -e PLUGIN_LIQUIBASE_URL='jdbc:postgresql://host:5432/db' \
  -e PLUGIN_LIQUIBASE_USERNAME='user' \
  -e PLUGIN_LIQUIBASE_PASSWORD='pass' \
  -e PLUGIN_LIQUIBASE_CHANGELOG_FILE='changelog.xml' \
  -e PLUGIN_LIQUIBASE_SEARCH_PATH='/liquibase/changelog' \
  drone-liquibase:local
```

## Code Style

- **Shell Scripts**: Use `shellcheck` for linting
- **Indentation**: 2 spaces for shell scripts
- **Quoting**: Always quote variables to handle spaces
- **Arrays**: Use bash arrays for command arguments to preserve spaces
- **Error Handling**: Use `set -e` in download scripts; explicit error handling in entrypoint

## Important Environment Variables

### Required

| Variable | Description |
|----------|-------------|
| `PLUGIN_COMMAND` | Liquibase command to execute (update, rollback, etc.) |

### Database Connection

| Variable | Description |
|----------|-------------|
| `PLUGIN_LIQUIBASE_URL` | JDBC connection URL |
| `PLUGIN_LIQUIBASE_USERNAME` | Database username |
| `PLUGIN_LIQUIBASE_PASSWORD` | Database password |
| `PLUGIN_LIQUIBASE_CHANGELOG_FILE` | Path to changelog file |
| `PLUGIN_LIQUIBASE_SEARCH_PATH` | Search path for changelog files |

### Harness Integration

| Variable | Description |
|----------|-------------|
| `GENERATE_STEP_OUTPUTS` | Enable step output generation (`true`/`false`) |
| `DRONE_OUTPUT` | Path to Drone output file |
| `PLUGIN_SUBSTITUTE_LIQUIBASE` | Base64+zstd encoded JSON for changelog substitutions |
| `PLUGIN_JSON_KEY` | Google Cloud service account key (for Spanner) |

### SSL/TLS Configuration

| Variable | Description |
|----------|-------------|
| Mount `/etc/ssl/certs/dbops/root_ca.crt` | Root CA certificate |
| Mount `/etc/ssl/certs/dbops/client.crt` | Client certificate (mTLS) |
| Mount `/etc/ssl/certs/dbops/client.key` | Client private key (mTLS) |

## Certificate Handling

The plugin auto-configures Java trust/key stores when certificates are mounted:

**TrustStore** (for server certificate validation):
1. Copies system cacerts to `/harness/certs/cacerts`
2. Imports root CA from `/etc/ssl/certs/dbops/root_ca.crt`
3. Sets `javax.net.ssl.trustStore` in JAVA_OPTS

**KeyStore** (for client certificate authentication):
1. Generates PKCS12 from client cert + key
2. Imports to `/harness/certs/jssecacerts`
3. Sets `javax.net.ssl.keyStore` in JAVA_OPTS
4. Cleans up temporary PKCS12 file

## Adding a New Liquibase Option

1. **Check if it's a global option**: Look at [Liquibase Global Options](https://docs.liquibase.com/concepts/connections/creating-config-properties.html)

2. **Add to whitelist**:
   ```bash
   echo "your-new-option" >> resources/global_options.txt
   ```

3. **Use in pipeline**:
   ```yaml
   settings:
     liquibase_your_new_option: "value"
   ```

## Adding a New Database Type

1. **Create download script** in `scripts/`:
   ```bash
   #!/usr/bin/env sh
   set -eux
   mkdir -p /liquibase/lib
   wget -O /liquibase/lib/your-driver.jar "https://..."
   ```

2. **Create Dockerfile** in `docker/`:
    - Use multi-stage build
    - Run download scripts in build stage
    - Copy JARs to final image

3. **Create manifest template** in `docker/`:
    - Define multi-arch image manifest

4. **Update CI/CD pipelines** to build new variant

## Release Process

1. **Make changes** and test locally
2. **Merge to main** in both:
    - Internal Harness Code repo
    - GitHub harness-community repo
3. **Create release tag**: Format `v<plugin_version>-<liquibase_version>` (e.g., `v1.8.0-4.33`)
4. **Run CI pipelines** to push to Docker Hub:
    - `harnesscommunity/drone-liquibase`
    - `plugins/drone-liquibase`

## Docker Image Locations

| Registry | Image |
|----------|-------|
| Harness Community | `harnesscommunity/drone-liquibase` |
| Drone Plugins | `plugins/drone-liquibase` |

## Common Gotchas

1. **Array vs String**: Use bash arrays for `command_args` to handle values with spaces correctly:
   ```bash
   command_args+=("--option" "$value")  # Correct
   command_args="$command_args --option $value"  # Wrong - breaks on spaces
   ```

2. **Exit Code Capture**: Use `PIPESTATUS[0]` to capture command exit code when piping to `tee`:
   ```bash
   "${command_args[@]}" 2>&1 | tee -a "$logfile"
   exit_code=${PIPESTATUS[0]}
   ```

3. **Environment Variable Cleanup**: Sensitive variables are unset after use (line 185) to prevent exposure.

4. **Non-Root Users**: The plugin supports non-root containers. Certificates are stored in `/harness/certs/` which is user-writable.

5. **Base Image Differences**: MongoDB images use UBI9 (Red Hat) base while others use Alpine. Library paths differ:
    - Alpine: `/usr/lib/libonig.so.5`
    - UBI9: `/usr/lib64/libonig.so.5`

## Debugging Tips

1. **Check command output**: The constructed command is echoed before execution (line 255)

2. **View logs**: Output is tee'd to a temp logfile (line 269)

3. **Check step outputs**: Look at `$DRONE_OUTPUT` for exit code and `/tmp/step_output.json` for detailed output

4. **SSL issues**: Verify certificates are readable and in correct format:
   ```bash
   openssl x509 -in /etc/ssl/certs/dbops/root_ca.crt -text -noout
   ```

5. **Test certificate import**:
   ```bash
   keytool -list -keystore /harness/certs/cacerts -storepass changeit
   ```

## Further Reading

| Document | Purpose |
|----------|---------|
| [README.md](README.md) | User-facing documentation |
| [Liquibase Docker Guide](https://docs.liquibase.com/workflows/liquibase-community/using-liquibase-and-docker.html) | Official Liquibase Docker usage |
| [dbops-extensions AGENTS.md](../dbopsExtensions/AGENTS.md) | DBOps extension development guide |
| [Drone Plugin Confluence](https://harness.atlassian.net/wiki/spaces/DB/pages/21827911819/Drone+Plugin) | Internal release process docs |

