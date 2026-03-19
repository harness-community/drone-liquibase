#!/usr/bin/env sh
set -eux

JCC_VERSION="12.1.4.0"

mkdir -p /liquibase/lib

# IBM DB2 JCC JDBC driver
wget -O /liquibase/lib/jcc-${JCC_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/ibm/db2/jcc/${JCC_VERSION}/jcc-${JCC_VERSION}.jar"
