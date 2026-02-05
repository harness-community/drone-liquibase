#!/bin/bash

# Check for required commands early
for cmd in jq zstd base64; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: Missing required command: $cmd"
    exit 1
  fi
done

# Function to read global options with retry mechanism
read_global_options() {
    local file_path="$1"
    local max_retries=3
    local retry_count=0
    local retry_delay=1
    local -n options_array="$2"  # Use nameref for array parameter

    while [ $retry_count -lt $max_retries ]; do
        if [ ! -r "$file_path" ]; then
            echo "Warning: File '$file_path' not readable, attempt $((retry_count + 1))/$max_retries"
            sleep $retry_delay
            retry_count=$((retry_count + 1))
            continue
        fi

        # Try to read the file
        if while IFS= read -r option || [ -n "$option" ]; do
            if [ -n "$option" ]; then
                options_array+=("$option")
            fi
        done < "$file_path" 2>/dev/null; then
            # Success - break out of retry loop
            break
        else
            echo "Warning: Failed to read file, attempt $((retry_count + 1))/$max_retries"
            sleep $retry_delay
            retry_count=$((retry_count + 1))
        fi
    done

    if [ $retry_count -eq $max_retries ]; then
        echo "Error: Failed to read '$file_path' after $max_retries attempts"
        return 1
    fi

    if [ ${#options_array[@]} -eq 0 ]; then
        echo "Warning: No options were read from '$file_path'"
    fi

    return 0
}

JAVA_HOME=$(java -XshowSettings:properties -version 2>&1 | grep 'java.home' | awk '{print $3}')

#if java_home is empty, then we use a different command. Fallback case
if [ -z "$JAVA_HOME" ]; then
  JAVA_HOME=$(dirname "$(dirname "$(readlink -f "$(which java)")")")
fi
export JAVA_HOME

#set common pwd
password="changeit"
USER_CERTS_DIR="/harness/certs"
USER_TRUSTSTORE="$USER_CERTS_DIR/cacerts"
USER_KEYSTORE="$USER_CERTS_DIR/jssecacerts"
SSL_CA_CERT_PATH="/etc/ssl/certs/dbops/root_ca.crt"
CLIENT_PKCS_12_PATH="$USER_CERTS_DIR/client.p12"
CLIENT_CERT="/etc/ssl/certs/dbops/client.crt"
CLIENT_KEY="/etc/ssl/certs/dbops/client.key"
CLIENT_CERT_ALIAS="client_pkcs12"

# Ensure cert directory exists with clear error handling for non-root or low-disk scenarios
if ! mkdir -p "$USER_CERTS_DIR" 2>/dev/null; then
    echo "Warning: Failed to create directory $USER_CERTS_DIR"
fi
chmod 700 "$USER_CERTS_DIR"

if [ -f "$SSL_CA_CERT_PATH" ]; then
    # Copy default truststore to user-writable location only if it doesn't already exist
    if [ -f "$USER_TRUSTSTORE" ]; then
        echo "Truststore already exists at $USER_TRUSTSTORE, skipping copying default trustStore"
    elif [ -f "${JAVA_HOME}/lib/security/cacerts" ]; then
        if ! cp "${JAVA_HOME}/lib/security/cacerts" "$USER_TRUSTSTORE" 2>/dev/null; then
            echo "Warning: Failed to copy system truststore from ${JAVA_HOME}/lib/security/cacerts"
            echo "Creating new empty truststore..."
        else
            echo "Copied system truststore to $USER_TRUSTSTORE"
        fi
    else
        echo "Warning: System truststore not found at ${JAVA_HOME}/lib/security/cacerts"
    fi

    # Check if the root CA certificate is already imported
    if keytool -list -keystore "$USER_TRUSTSTORE" -storepass "$password" -alias mongodb-root-ca-cert >/dev/null 2>&1; then
        echo "Root CA certificate already exists in truststore, skipping import"
    else
        echo "Importing self signed certificate into trustStore..."
        if [ -z "$JAVA_HOME" ]; then
            echo "Warning: JAVA_HOME is not set. Cannot import self signed certificate in path $SSL_CA_CERT_PATH."
        fi
        if ! keytool -importcert \
            -alias mongodb-root-ca-cert \
            -keystore "$USER_TRUSTSTORE" \
            -storepass "$password" \
            -trustcacerts \
            -file "$SSL_CA_CERT_PATH" \
            -noprompt 2>&1; then
            echo "Warning: Failed to import root CA certificate"
            echo "File: $SSL_CA_CERT_PATH"
        else
            echo "Successfully imported root CA certificate into $USER_TRUSTSTORE"
        fi
    fi
    # Append truststore settings to JAVA_OPTS
    JAVA_OPTS="${JAVA_OPTS:+$JAVA_OPTS }-Djavax.net.ssl.trustStore=$USER_TRUSTSTORE"
    JAVA_OPTS="${JAVA_OPTS:+$JAVA_OPTS }-Djavax.net.ssl.trustStorePassword=$password"
fi

if [ -f "$CLIENT_CERT" ]; then
  if [ -f "$CLIENT_KEY" ]; then

        # Check if keystore already exists and contains the client certificate
        if [ -f "$USER_KEYSTORE" ] && keytool -list -keystore "$USER_KEYSTORE" -storepass "$password" -alias "$CLIENT_CERT_ALIAS" >/dev/null 2>&1; then
            echo "Client certificate already exists in keystore at $USER_KEYSTORE, skipping import"
        else
            echo "generating pkcs12 based on $CLIENT_CERT and $CLIENT_KEY"
            openssl pkcs12 -export -in "$CLIENT_CERT" -inkey "$CLIENT_KEY" -out "$CLIENT_PKCS_12_PATH" -name "$CLIENT_CERT_ALIAS" -password pass:"$password"

            echo "Importing client certificate into keyStore..."
            if [ -z "$JAVA_HOME" ]; then
                echo "Warning: JAVA_HOME is not set. Cannot import client certificate for $CLIENT_CERT and $CLIENT_KEY"
            fi
            if ! keytool -importkeystore \
                -destkeystore "$USER_KEYSTORE" \
                -srckeystore "$CLIENT_PKCS_12_PATH" \
                -srcstoretype PKCS12 \
                -alias "$CLIENT_CERT_ALIAS" \
                -storepass "$password" \
                -srcstorepass "$password" \
                -noprompt 2>&1; then
                echo "Warning: Failed to import client certificate into keystore"
                echo "Source: $CLIENT_PKCS_12_PATH"
                echo "Destination: $USER_KEYSTORE"
            else
                echo "Successfully imported client certificate into $USER_KEYSTORE"
            fi

            # Clean up sensitive PKCS12 file now that it is imported
            if [ -f "$CLIENT_PKCS_12_PATH" ]; then
                rm -f "$CLIENT_PKCS_12_PATH"
            fi
        fi

        # Append keystore settings to JAVA_OPTS (truststore already appended above if present)
        JAVA_OPTS="${JAVA_OPTS:+$JAVA_OPTS }-Djavax.net.ssl.keyStore=$USER_KEYSTORE"
        JAVA_OPTS="${JAVA_OPTS:+$JAVA_OPTS }-Djavax.net.ssl.keyStorePassword=$password"
  fi
fi

if [ -n "$JAVA_OPTS" ]; then
    export JAVA_OPTS
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

# Read global options with retry mechanism
if ! read_global_options "$global_options_file" global_options; then
    exit 1
fi

# Initialize cleanup actions array
declare -a cleanup_actions
cleanup() {
    for action in "${cleanup_actions[@]}"; do
        eval "$action" 2>/dev/null || true
    done
}
# it will execute whatever is in cleanup_actions at exit time
trap cleanup EXIT

# Kerberos Authentication Support, if Kerberos is enabled (PLUGIN_KERBEROS_USER_PRINCIPAL must be set)
if [ -n "$PLUGIN_KERBEROS_USER_PRINCIPAL" ]; then

    echo "Initiating kerberos authentication for principal: $PLUGIN_KERBEROS_USER_PRINCIPAL"
    if [ -n "$PLUGIN_KERBEROS_PASSWORD" ]; then
        if ! echo "$PLUGIN_KERBEROS_PASSWORD" | kinit -f "$PLUGIN_KERBEROS_USER_PRINCIPAL"; then
            echo "Error: Password-Based Kerberos authentication kinit failed"
            exit 1
        fi
    elif [ -n "$PLUGIN_KERBEROS_KEYTAB_FILE_PATH" ]; then
        if [ ! -f "$PLUGIN_KERBEROS_KEYTAB_FILE_PATH" ]; then
            echo "Error: Keytab file not found: $PLUGIN_KERBEROS_KEYTAB_FILE_PATH"
            exit 1
        fi
        if ! kinit -f -k -t "$PLUGIN_KERBEROS_KEYTAB_FILE_PATH" "$PLUGIN_KERBEROS_USER_PRINCIPAL"; then
            echo "Error: Keytab-Based Kerberos authentication kinit failed"
            exit 1
        fi
    else
        echo "Error: PLUGIN_KERBEROS_USER_PRINCIPAL requires an authentication method. Set either PLUGIN_KERBEROS_PASSWORD or PLUGIN_KERBEROS_KEYTAB_FILE_PATH"
        exit 1
    fi
    echo "Kerberos authentication successful for principal: $PLUGIN_KERBEROS_USER_PRINCIPAL"
    cleanup_actions+=('kdestroy 2>/dev/null')

    # Set Kerberos configuration file path for Java
    JAVA_OPTS="${JAVA_OPTS:+$JAVA_OPTS }-Doracle.net.authentication_services=KERBEROS5"
    export JAVA_OPTS
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
    cleanup_actions+=("rm -f \"$substitute_properties_decoded\"")

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

# Check if the file exists and remove before running the command
if [ -f "$STEP_OUTPUT_FILE" ]; then
  rm -rf $STEP_OUTPUT_FILE
fi

# Execute the command using the array format
# we are using an array to ensure that values containing spaces are preserved
# Without an array, each word within a space is considered as a liquibase command
{
  "${command_args[@]}" 2>&1 | tee -a "$logfile"
  exit_code=${PIPESTATUS[0]}  # Capture the exit code of the actual command
}

echo "exit_code=$exit_code" > "$DRONE_OUTPUT"

STEP_OUTPUT_FILE="/tmp/step_output.json"
# Check if the file exists and update DRONE_OUTPUT accordingly

if [ "$GENERATE_STEP_OUTPUTS" = "true" ]; then
    if [ -f "$STEP_OUTPUT_FILE" ]; then
      # Read the entire contents of the JSON file into 'step_output'
      step_output=$(cat "$STEP_OUTPUT_FILE")
      echo "step_output=${step_output}" >> "$DRONE_OUTPUT"
      rm -rf $STEP_OUTPUT_FILE
    fi
fi