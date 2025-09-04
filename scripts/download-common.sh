#!/usr/bin/env sh
set -eux

LIQUIBASE_DBOPS_EXT_VERSION="1.13.1"
LIQUIBASE_DBOPS_EXT_ZSTD_VERSION="1.5.5-5"
OKIO_VERSION="3.2.0"
OKHTTP_VERSION="4.11.0"
LOGGING_INTERCEPTOR_VERSION="4.11.0"
RETROFIT_VERSION="3.0.0"
CONVERTOR_GSON_VERSION="3.0.0"
GSON_VERSION="2.13.1"
KOTLIN_STLIB_VERSION="2.1.21"
BOUNCY_CASTLE_VERSION="1.78.1"


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
