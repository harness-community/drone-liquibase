#!/usr/bin/env sh
set -eux

# Download CloudSQL Socket Factory shaded JAR from Google Artifact Registry
# This JAR contains all Cloud SQL dependencies with properly merged SPI files

# Version format: <internal_version>-<cloudsql_base_version>
# Example: 1.0.0-1.28.1 where 1.0.0 is internal version, 1.28.1 is Cloud SQL Socket Factory version

INTERNAL_VERSION="1.0.0"
CLOUDSQL_BASE_VERSION="1.28.1"
CLOUDSQL_VERSION="${INTERNAL_VERSION}-${CLOUDSQL_BASE_VERSION}"

GAR_URL="https://us-maven.pkg.dev/gar-prod-setup/harness-maven-public/io/harness/cloudsql-socket-factory/${CLOUDSQL_VERSION}/cloudsql-socket-factory-${CLOUDSQL_VERSION}.jar"

mkdir -p /liquibase/lib

wget -q -O /liquibase/lib/cloudsql-socket-factory.jar "$GAR_URL"

echo "Downloaded cloudsql-socket-factory-${CLOUDSQL_VERSION}.jar"
