#!/usr/bin/env sh
set -eux

LIQUIBASE_DBOPS_EXT_VERSION="1.21.0"
LIQUIBASE_DBOPS_EXT_ZSTD_VERSION="1.5.5-5"
OKIO_VERSION="3.2.0"
OKHTTP_VERSION="4.11.0"
LOGGING_INTERCEPTOR_VERSION="4.11.0"
RETROFIT_VERSION="3.0.0"
CONVERTOR_GSON_VERSION="3.0.0"
GSON_VERSION="2.13.1"
KOTLIN_STLIB_VERSION="2.1.21"
BOUNCY_CASTLE_VERSION="1.78.1"
FAILSAFE_VERSION="2.4.4"
GUAVA_VERSION="33.4.0-jre"
MSSQL_JDBC_VERSION="12.10.2.jre11"
MYSQL_CONNECTOR_VERSION="9.6.0"
JACKSON_VERSION="2.18.6"
COMMONS_CODEC_VERSION="1.21.0"
POSTGRESQL_JDBC_VERSION="42.7.11"


mkdir -p /liquibase/lib

# Core dbops extension jars
wget -O /liquibase/lib/dbops-extensions-${LIQUIBASE_DBOPS_EXT_VERSION}.jar \
  "https://us-maven.pkg.dev/gar-prod-setup/harness-maven-public/io/harness/dbops-extensions/${LIQUIBASE_DBOPS_EXT_VERSION}/dbops-extensions-${LIQUIBASE_DBOPS_EXT_VERSION}.jar"

# zstd-jni jar
wget -O /liquibase/lib/zstd-jni-${LIQUIBASE_DBOPS_EXT_ZSTD_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/github/luben/zstd-jni/${LIQUIBASE_DBOPS_EXT_ZSTD_VERSION}/zstd-jni-${LIQUIBASE_DBOPS_EXT_ZSTD_VERSION}.jar"

# okio & okio-jvm
wget -O /liquibase/lib/okio-${OKIO_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/squareup/okio/okio/${OKIO_VERSION}/okio-${OKIO_VERSION}.jar"
wget -O /liquibase/lib/okio-jvm-${OKIO_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/squareup/okio/okio-jvm/${OKIO_VERSION}/okio-jvm-${OKIO_VERSION}.jar"

# okhttp & logging-interceptor
wget -O /liquibase/lib/okhttp-${OKHTTP_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/squareup/okhttp3/okhttp/${OKHTTP_VERSION}/okhttp-${OKHTTP_VERSION}.jar"
wget -O /liquibase/lib/logging-interceptor-${LOGGING_INTERCEPTOR_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/squareup/okhttp3/logging-interceptor/${LOGGING_INTERCEPTOR_VERSION}/logging-interceptor-${LOGGING_INTERCEPTOR_VERSION}.jar"

# retrofit & converter-gson & gson & kotlin-stdlib
wget -O /liquibase/lib/retrofit-${RETROFIT_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/squareup/retrofit2/retrofit/${RETROFIT_VERSION}/retrofit-${RETROFIT_VERSION}.jar"
wget -O /liquibase/lib/converter-gson-${CONVERTOR_GSON_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/squareup/retrofit2/converter-gson/${CONVERTOR_GSON_VERSION}/converter-gson-${CONVERTOR_GSON_VERSION}.jar"
wget -O /liquibase/lib/gson-${GSON_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/google/code/gson/gson/${GSON_VERSION}/gson-${GSON_VERSION}.jar"
wget -O /liquibase/lib/kotlin-stdlib-${KOTLIN_STLIB_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-stdlib/${KOTLIN_STLIB_VERSION}/kotlin-stdlib-${KOTLIN_STLIB_VERSION}.jar"

# Bouncy Castle
wget -O /liquibase/lib/bcpkix-jdk18on-${BOUNCY_CASTLE_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/bouncycastle/bcpkix-jdk18on/${BOUNCY_CASTLE_VERSION}/bcpkix-jdk18on-${BOUNCY_CASTLE_VERSION}.jar"
wget -O /liquibase/lib/bcprov-jdk18on-${BOUNCY_CASTLE_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/bouncycastle/bcprov-jdk18on/${BOUNCY_CASTLE_VERSION}/bcprov-jdk18on-${BOUNCY_CASTLE_VERSION}.jar"

#jodah failsafe retry
wget -O /liquibase/lib/failsafe-${FAILSAFE_VERSION}.jar \
  "https://repo1.maven.org/maven2/net/jodah/failsafe/${FAILSAFE_VERSION}/failsafe-${FAILSAFE_VERSION}.jar"

# Guava
wget -O /liquibase/lib/guava-${GUAVA_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/google/guava/guava/${GUAVA_VERSION}/guava-${GUAVA_VERSION}.jar"

# mssql-jdbc (replaces vulnerable 12.10.1 bundled in base image)
wget -O /liquibase/lib/mssql-jdbc-${MSSQL_JDBC_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/microsoft/sqlserver/mssql-jdbc/${MSSQL_JDBC_VERSION}/mssql-jdbc-${MSSQL_JDBC_VERSION}.jar"

# postgresql JDBC driver (replaces vulnerable 42.7.7 bundled in liquibase base image — CVE-2026-42198)
wget -O /liquibase/lib/postgresql-${POSTGRESQL_JDBC_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/postgresql/postgresql/${POSTGRESQL_JDBC_VERSION}/postgresql-${POSTGRESQL_JDBC_VERSION}.jar"

# mysql-connector-j (replaces lpm add mysql)
wget -O /liquibase/lib/mysql-connector-j-${MYSQL_CONNECTOR_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/mysql/mysql-connector-j/${MYSQL_CONNECTOR_VERSION}/mysql-connector-j-${MYSQL_CONNECTOR_VERSION}.jar"

# jackson-databind
wget -O /liquibase/lib/jackson-databind-${JACKSON_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-databind/${JACKSON_VERSION}/jackson-databind-${JACKSON_VERSION}.jar"

# jackson-core
wget -O /liquibase/lib/jackson-core-${JACKSON_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-core/${JACKSON_VERSION}/jackson-core-${JACKSON_VERSION}.jar"

# jackson-annotations
wget -O /liquibase/lib/jackson-annotations-${JACKSON_VERSION}.jar \
  "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-annotations/${JACKSON_VERSION}/jackson-annotations-${JACKSON_VERSION}.jar"

# commons-codec
wget -O /liquibase/lib/commons-codec-${COMMONS_CODEC_VERSION}.jar \
  "https://repo1.maven.org/maven2/commons-codec/commons-codec/${COMMONS_CODEC_VERSION}/commons-codec-${COMMONS_CODEC_VERSION}.jar"
