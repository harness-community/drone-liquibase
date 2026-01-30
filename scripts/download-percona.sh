#!/usr/bin/env sh
set -eux

PERCONA_TOOLKIT_VERSION="4.33.0"

mkdir -p /liquibase/lib

# Percona Toolkit extension JAR
wget -O /liquibase/lib/liquibase-percona-${PERCONA_TOOLKIT_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/liquibase/ext/liquibase-percona/${PERCONA_TOOLKIT_VERSION}/liquibase-percona-${PERCONA_TOOLKIT_VERSION}.jar"
