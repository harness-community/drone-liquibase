#!/usr/bin/env sh
set -eux

MONGO_DRIVER_CORE_VERSION="5.0.0"
MONGO_DRIVER_SYNC_VERSION="5.0.0"
BSON_VERSION="5.0.0"
LIQUIBASE_MONGODB_DBOPS_EXT_VERSION="1.1.0-4.24.0"
SNAKEYAML_VERSION="2.2"


mkdir -p /liquibase/lib

# MongoDB-specific jars
wget -O /liquibase/lib/mongodb-driver-core-${MONGO_DRIVER_CORE_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/mongodb/mongodb-driver-core/${MONGO_DRIVER_CORE_VERSION}/mongodb-driver-core-${MONGO_DRIVER_CORE_VERSION}.jar"
wget -O /liquibase/lib/mongodb-driver-sync-${MONGO_DRIVER_SYNC_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/mongodb/mongodb-driver-sync/${MONGO_DRIVER_SYNC_VERSION}/mongodb-driver-sync-${MONGO_DRIVER_SYNC_VERSION}.jar"
wget -O /liquibase/lib/bson-${BSON_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/mongodb/bson/${BSON_VERSION}/bson-${BSON_VERSION}.jar"
wget -O /liquibase/lib/liquibase-mongodb-dbops-extension-${LIQUIBASE_MONGODB_DBOPS_EXT_VERSION}.jar \
  "https://us-maven.pkg.dev/gar-prod-setup/harness-maven-public/io/harness/liquibase-mongodb-dbops-extension/${LIQUIBASE_MONGODB_DBOPS_EXT_VERSION}/liquibase-mongodb-dbops-extension-${LIQUIBASE_MONGODB_DBOPS_EXT_VERSION}.jar"
wget -O /liquibase/lib/snakeyaml-${SNAKEYAML_VERSION}.jar \
  "https://repo1.maven.org/maven2/org/yaml/snakeyaml/${SNAKEYAML_VERSION}/snakeyaml-${SNAKEYAML_VERSION}.jar"
