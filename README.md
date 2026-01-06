# drone-liquibase
# Overview
A Drone plugin for Liquibase built on top of liquibase/liquibase official docker image for database schema version control and change management

# Supported tags and respective Dockerfile links
- `latest`

# How to use this image
```
docker run --rm -v <PATH TO CHANGELOG DIR>:/liquibase/changelog
-e PLUGIN_LIQUIBASE_URL="jdbc:sqlserver://<IP OR HOSTNAME>:1433;database=<DATABASE>;"
-e PLUGIN_LIQUIBASE_CHANGELOG_FILE=com/example/changelog.xml
-e PLUGIN_LIQUIBASE_USERNAME=<USERNAME>
-e PLUGIN_LIQUIBASE_PASSWORD=<PASSWORD>
-e PLUGIN_SEARCH_PATH='/liquibase/changelog'
-e PLUGIN_COMMAND='update'
harnesscommunity/drone-liquibase:latest
```
PLUGIN_COMMAND: This specifies the liquibase command to run. The above example runs the liquibase update command

## SSL Configuration for Non-Root Users

This plugin supports SSL connections with both root and non-root users. Certificates are stored in user-writable locations.

### Directory Structure
```
/harness/certs/        # User-writable certificate directory
├── cacerts            # TrustStore (auto-generated from system certs)
├── jssecacerts        # KeyStore (if client certificate auth is used)
```

### Mount Your Certificates
```
docker run --rm \
  --user 1000:1000 \
  -v /path/to/certs:/etc/ssl/certs/dbops:ro \
  -e PLUGIN_LIQUIBASE_URL="jdbc:mongodb://host:27017/db?ssl=true" \
  harnesscommunity/drone-liquibase:latest
```

### Required Files
- Root CA Certificate: `/etc/ssl/certs/dbops/root_ca.crt` (required for self-signed certs)
- Client Certificate (optional): `/etc/ssl/certs/dbops/client.crt`
- Client Key (optional): `/etc/ssl/certs/dbops/client.key`
