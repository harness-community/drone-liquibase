#!/bin/bash

# Check for required commands early
for cmd in jq zstd base64; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: Missing required command: $cmd"
    exit 1
  fi
done

JAVA_HOME=$(java -XshowSettings:properties -version 2>&1 | grep 'java.home' | awk '{print $3}')

#if java_home is empty, then we use a different command. Fallback case
if [ -z "$JAVA_HOME" ]; then
  JAVA_HOME=$(dirname "$(dirname "$(readlink -f "$(which java)")")")
fi
export JAVA_HOME

#set common pwd
password="changeit"

# Check if Root CA file exists and import it
SSL_CA_CERT_PATH="/etc/ssl/certs/dbops/root_ca.crt"
if [ -f "$SSL_CA_CERT_PATH" ]; then
    echo "Importing self signed certificate into default JVM trustStore..."
    if [ -z "$JAVA_HOME" ]; then
        echo "Error: JAVA_HOME is not set. Cannot import self signed certificate in path $SSL_CA_CERT_PATH."
        exit 1
    fi
    keytool -importcert \
    -alias mongodb-root-ca-cert \
    -keystore "${JAVA_HOME}/lib/security/cacerts" \
    -storepass "$password" \
    -trustcacerts \
    -file "$SSL_CA_CERT_PATH" \
    -noprompt
        #JAVA_OPTS is a variable that belongs to liquibase. This sets the env variables
    export JAVA_OPTS="-Djavax.net.ssl.trustStore=$JAVA_HOME/lib/security/cacerts -Djavax.net.ssl.trustStorePassword=$password"
fi


# Check if client certificate key file exists and import it
CLIENT_PKCS_12_PATH="/etc/ssl/certs/dbops/client.p12"
CLIENT_CERT="/etc/ssl/certs/dbops/client.crt"
CLIENT_KEY="/etc/ssl/certs/dbops/client.key"

if [ -f "$CLIENT_CERT" ]; then
  if [ -f "$CLIENT_KEY" ]; then

        echo "generating pkcs12 based on $CLIENT_CERT and $CLIENT_KEY"
        openssl pkcs12 -export -in "$CLIENT_CERT" -inkey "$CLIENT_KEY" -out "$CLIENT_PKCS_12_PATH" -name client_pkcs12 -password pass:"$password"

        echo "Importing client certificate into default JVM keyStore..."
        if [ -z "$JAVA_HOME" ]; then
            echo "Error: JAVA_HOME is not set. Cannot import client certificate for $CLIENT_CERT and $CLIENT_KEY"
            exit 1
        fi
        keytool -importkeystore \
        -destkeystore "${JAVA_HOME}/lib/security/jssecacerts" \
        -srckeystore "$CLIENT_PKCS_12_PATH" \
        -srcstoretype PKCS12 \
        -alias client_pkcs12 \
        -storepass "$password" \
        -srcstorepass "$password"
        #JAVA_OPTS is a variable that belongs to liquibase. This sets the env variables
        export JAVA_OPTS="-Djavax.net.ssl.keyStore=$JAVA_HOME/lib/security/jssecacerts -Djavax.net.ssl.keyStorePassword=$password -Djavax.net.ssl.trustStore=$JAVA_HOME/lib/security/cacerts -Djavax.net.ssl.trustStorePassword=$password"
  fi
fi

# Check if PLUGIN_COMMAND is non-empty
if [ -z "$PLUGIN_COMMAND" ]; then
    echo "Error: PLUGIN_COMMAND is empty. Please set PLUGIN_COMMAND before running the script."
    exit 1
fi

# Read global options from file
global_options_file="/resources/global_options.txt"

# Initialize an array to store global options
declare -a global_options

# Maximum number of retries
max_retries=3
retry_count=0
retry_delay=1

while [ $retry_count -lt $max_retries ]; do
    if [ ! -r "$global_options_file" ]; then
        echo "Warning: File '$global_options_file' not readable, attempt $((retry_count + 1))/$max_retries"
        sleep $retry_delay
        retry_count=$((retry_count + 1))
        continue
    fi

    # Try to read the file
    if while IFS= read -r option || [ -n "$option" ]; do
        if [ -n "$option" ]; then
            global_options+=("$option")
        fi
    done < "$global_options_file" 2>/dev/null; then
        # Success - break out of retry loop
        break
    else
        echo "Warning: Failed to read file, attempt $((retry_count + 1))/$max_retries"
        sleep $retry_delay
        retry_count=$((retry_count + 1))
    fi
done

if [ $retry_count -eq $max_retries ]; then
    echo "Error: Failed to read '$global_options_file' after $max_retries attempts"
    exit 1
fi

if [ ${#global_options[@]} -eq 0 ]; then
    echo "Warning: No options were read from '$global_options_file'"
fi

# Initialize an array to hold the constructed argument list
# We are using an array to ensure that values containing spaces are preserved
# Without an array, each word within a space is considered as a liquibase command
declare -a command_args

# Iterate through the list of global options
for option in "${global_options[@]}"; do
    env_var_name="PLUGIN_LIQUIBASE_$(echo "$option" | tr '-' '_' | tr '[:lower:]' '[:upper:]')"

    # If the environment variable is set, append to the argument list
    if [ -n "${!env_var_name}" ]; then
        command_args+=("--$option" "${!env_var_name}")
        # unset the environment variable to hide it from now on
        unset "$env_var_name"
    fi
done

command_args+=("$PLUGIN_COMMAND")

# Add changelog substitution properties
if [ -n "$PLUGIN_SUBSTITUTE_LIQUIBASE" ]; then
    # Step 1: Create temporary file for decoded content
    substitute_properties_decoded=$(mktemp)
    trap 'rm -f "$substitute_properties_decoded"' EXIT

    # Step 2: Base64 decode directly to file
    if ! echo "$PLUGIN_SUBSTITUTE_LIQUIBASE" | base64 -d > "$substitute_properties_decoded" 2>/dev/null; then
        echo "Error: Failed to decode base64 input"
        exit 1
    fi
    
    # Step 3: Decompress using zstd
    if ! decompressed=$(zstd -d -c "$substitute_properties_decoded"); then
        echo "Error: zstd decompression failed"
        exit 1
    fi
    
    # Check for empty decompressed data
    if [ -z "$decompressed" ]; then
        echo "Error: Decompressed data is empty"
        exit 1
    fi
    
    # Validate JSON format
    if ! echo "$decompressed" | jq empty 2>/dev/null; then
        echo "Error: Invalid JSON in decompressed input"
        exit 1
    fi
    
    # Step 4: Parse JSON and convert to Liquibase arguments
    while IFS= read -r arg; do
        if [ -n "$arg" ]; then
            command_args+=("$arg")
        fi
    done < <(echo "$decompressed" | jq -r 'to_entries | .[] | "-D\(.key)=\(.value)"')
fi

# Add remaining environment variables
for var in $(env | grep '^PLUGIN_LIQUIBASE_' | awk -F= '{print $1}'); do
    # Remove "PLUGIN_LIQUIBASE_" from the variable name, convert to lowercase, and replace underscores with hyphens
    var_name="${var#PLUGIN_LIQUIBASE_}"
    var_name_lower="$(echo "$var_name" | tr '[:upper:]' '[:lower:]' | tr '_' '-')"
    value="${!var}"

    # Append the argument
    command_args+=("--$var_name_lower" "$value")
done

# Define the SA target file
SERVICE_ACCOUNT_KEY_FILE="/tmp/harness-google-application-credentials.json"

# Check if the environment variable is set and the file does not already exist
if [[ -n "$PLUGIN_JSON_KEY" && ! -f "$SERVICE_ACCOUNT_KEY_FILE" ]]; then
    echo "Creating service account key file..."

    # Write the content of PLUGIN_JSON_KEY to the file if it is set
    echo "${PLUGIN_JSON_KEY:-}" > "$SERVICE_ACCOUNT_KEY_FILE"
    # Export the GOOGLE_APPLICATION_CREDENTIALS variable to point to the file
    export GOOGLE_APPLICATION_CREDENTIALS="$SERVICE_ACCOUNT_KEY_FILE"
fi

# Print the constructed command for debugging
command_args=("/liquibase/liquibase" "${command_args[@]}")
echo "${command_args[@]}"

# Create unique file to avoid override in parallel steps
logfile=$(mktemp)

# Execute the command using the array format
# we are using an array to ensure that values containing spaces are preserved
# Without an array, each word within a space is considered as a liquibase command
{
  "${command_args[@]}" 2>&1 | tee -a "$logfile"
  exit_code=${PIPESTATUS[0]}  # Capture the exit code of the actual command
}

encoded_command_logs=$(cat "$logfile" | base64 -w 0)

encoded_command_logs=`echo $encoded_command_logs | tr = -`

echo "encoded_command_logs=$encoded_command_logs" > "$DRONE_OUTPUT"
echo "exit_code=$exit_code" >> "$DRONE_OUTPUT"
