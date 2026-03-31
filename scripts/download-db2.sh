#!/usr/bin/env sh
set -eux

JCC_VERSION="12.1.4.0"
JT400_VERSION="21.0.6"
LIQUIBASE_DB2I_VERSION="4.33.0"

mkdir -p /liquibase/lib

# IBM DB2 JCC JDBC driver (DB2 LUW)
wget -O /liquibase/lib/jcc-${JCC_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/ibm/db2/jcc/${JCC_VERSION}/jcc-${JCC_VERSION}.jar"

# IBM JTOpen JDBC driver (DB2 for iSeries / AS400)
wget -O /liquibase/lib/jt400-${JT400_VERSION}-java11.jar \
  "https://repo1.maven.org/maven2/net/sf/jt400/jt400/${JT400_VERSION}/jt400-${JT400_VERSION}-java11.jar"

# Liquibase DB2 iSeries extension
wget -O /liquibase/lib/liquibase-db2i-${LIQUIBASE_DB2I_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/liquibase/ext/liquibase-db2i/${LIQUIBASE_DB2I_VERSION}/liquibase-db2i-${LIQUIBASE_DB2I_VERSION}.jar"
