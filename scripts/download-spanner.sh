#!/usr/bin/env sh
set -eux

LIQUIBASE_SPANNER_VERSION="4.30.0.1"

mkdir -p /liquibase/lib

# Spanner-specific jar
wget -O /liquibase/lib/liquibase-spanner-${LIQUIBASE_SPANNER_VERSION}-all.jar \
  "https://github.com/cloudspannerecosystem/liquibase-spanner/releases/download/${LIQUIBASE_SPANNER_VERSION}/liquibase-spanner-${LIQUIBASE_SPANNER_VERSION}-all.jar"
