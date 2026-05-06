#!/usr/bin/env sh
# aws-advanced-jdbc-wrapper only (used by docker/Dockerfile.linux.aws-rds.*). IAM SDK JARs come from Maven stage (maven/pom-aws-jdbc-iam.xml).
set -eux

AWS_ADVANCED_JDBC_WRAPPER_VERSION="3.3.0"
mkdir -p /liquibase/lib

wget -O "/liquibase/lib/aws-advanced-jdbc-wrapper-${AWS_ADVANCED_JDBC_WRAPPER_VERSION}.jar" \
  "https://repo1.maven.org/maven2/software/amazon/jdbc/aws-advanced-jdbc-wrapper/${AWS_ADVANCED_JDBC_WRAPPER_VERSION}/aws-advanced-jdbc-wrapper-${AWS_ADVANCED_JDBC_WRAPPER_VERSION}.jar"
