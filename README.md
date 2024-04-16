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
