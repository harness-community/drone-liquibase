#!/usr/bin/env sh
set -eux

LIQUIBASE_BIGQUERY_VERSION="4.33.0"
SIMBA_JDBC_VERSION="1.6.5.1002"

mkdir -p /liquibase/lib

# Open-source Liquibase BigQuery extension (registers jdbc:bigquery driver with Liquibase)
wget -O /liquibase/lib/liquibase-bigquery-${LIQUIBASE_BIGQUERY_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/liquibase/ext/liquibase-bigquery/${LIQUIBASE_BIGQUERY_VERSION}/liquibase-bigquery-${LIQUIBASE_BIGQUERY_VERSION}.jar"
echo "Downloaded liquibase-bigquery-${LIQUIBASE_BIGQUERY_VERSION}.jar"

# Simba BigQuery JDBC driver and its runtime dependencies (distributed by Google)
wget -O /tmp/simba-bigquery.zip \
  "https://storage.googleapis.com/simba-bq-release/jdbc/SimbaJDBCDriverforGoogleBigQuery42_${SIMBA_JDBC_VERSION}.zip"
echo "Downloaded Simba BigQuery JDBC driver zip"

mkdir -p /tmp/simba
cd /tmp/simba
unzip -j /tmp/simba-bigquery.zip '*.jar'

# Remove JARs that duplicate libraries already provided by download-common.sh
# or the base Liquibase image, to avoid classpath version conflicts.
rm -f guava-*.jar
rm -f gson-*.jar
rm -f jackson-*.jar
rm -f commons-codec-*.jar
rm -f slf4j-*.jar

cp *.jar /liquibase/lib/
echo "Installed Simba JDBC driver and $(ls *.jar | wc -l) dependency JARs"

rm -rf /tmp/simba /tmp/simba-bigquery.zip
