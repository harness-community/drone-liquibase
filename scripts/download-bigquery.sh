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

# Remove JARs from the Simba bundle that duplicate libraries already provided
# by download-common.sh or the base Liquibase image, to avoid classpath
# version conflicts.
# We read download-common.sh to build the list automatically: first eval its
# version variables, then extract every "wget -O /liquibase/lib/<name>.jar"
# target, expand the variables to get real filenames, strip the version suffix
# (everything from the first "-<digit>" onward) to get artifact prefixes,
# and remove matching Simba JARs. This way any JAR added to download-common.sh
# in the future is picked up here with zero changes.
COMMON_SCRIPT="/scripts/download-common.sh"
eval "$(grep '_VERSION=' "$COMMON_SCRIPT")"

for target in $(grep 'wget -O /liquibase/lib/' "$COMMON_SCRIPT" \
                | sed 's/.*wget -O //; s/ .*//' | tr -d '\\'); do
  expanded=$(eval echo "$target")
  prefix=$(basename "$expanded" .jar | sed 's/-[0-9].*//')
  rm -f "${prefix}"-*.jar
done

# slf4j is bundled in the base Liquibase image, not in download-common.sh
rm -f slf4j-*.jar

cp *.jar /liquibase/lib/
echo "Installed Simba JDBC driver and $(ls *.jar | wc -l) dependency JARs"

rm -rf /tmp/simba /tmp/simba-bigquery.zip
