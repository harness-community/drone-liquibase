#!/usr/bin/env sh
set -eux

# Download CloudSQL Socket Factory shaded JAR from Google Artifact Registry
# This JAR contains all Cloud SQL dependencies with properly merged SPI files

CLOUDSQL_VERSION="1.28.2"
GAR_URL="https://us-maven.pkg.dev/gar-prod-setup/harness-maven-public/io/harness/cloudsql-socket-factory/${CLOUDSQL_VERSION}/cloudsql-socket-factory-${CLOUDSQL_VERSION}.jar"

mkdir -p /liquibase/lib

wget -q -O /liquibase/lib/cloudsql-socket-factory.jar "$GAR_URL"

echo "Downloaded cloudsql-socket-factory-${CLOUDSQL_VERSION}.jar"
