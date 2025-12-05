#!/bin/sh
set -eux

# --- Logic to determine architecture ---
# Read the architecture from the BUILD_ARCH environment variable.
# Fail if the variable is not set.
if [ -z "${BUILD_ARCH}" ]; then
  echo "Error: The BUILD_ARCH environment variable is not set. Please set it before running."
  exit 1
fi

detected_arch="${BUILD_ARCH}"
echo "Using architecture from BUILD_ARCH env var: ${detected_arch}"

# Normalize the architecture name for the download URL
case "$detected_arch" in
  "x86_64" | "amd64") ARCH="x64" ;;
  "aarch64" | "arm64") ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $detected_arch"
    exit 1
    ;;
esac

# --- Download and Install ---
MONGOSH_VERSION="2.2.10"
URL="https://downloads.mongodb.com/compass/mongosh-${MONGOSH_VERSION}-linux-${ARCH}.tgz"

echo "Downloading mongosh for ${ARCH} from ${URL}..."
curl -fL "${URL}" -o /tmp/mongosh.tgz

echo "Extracting archive..."
tar xzf /tmp/mongosh.tgz -C /tmp

echo "Installing mongosh to /usr/local/bin..."
mv "/tmp/mongosh-${MONGOSH_VERSION}-linux-${ARCH}/bin/mongosh" /usr/local/bin/mongosh
chmod +x /usr/local/bin/mongosh

echo "Cleaning up temporary files..."
rm -rf /tmp/mongosh.tgz "/tmp/mongosh-${MONGOSH_VERSION}-linux-${ARCH}"

echo "Verifying mongosh installation..."
mongosh --version
echo "mongosh installed successfully."
