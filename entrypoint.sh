#!/bin/bash

# Read global options from file
global_options_file="/resources/global_options.txt"

# Check if the file exists
if [ ! -f "$global_options_file" ]; then
    echo "Error: File '$global_options_file' not found."
    exit 1
fi

# Initialize an array to store global options
declare -a global_options

# Read global options into an array
while IFS= read -r option || [[ -n "$option" ]]; do
    global_options+=("$option")
done < "$global_options_file"

# Check if PLUGIN_COMMAND is non-empty
if [ -z "$PLUGIN_COMMAND" ]; then
    echo "Error: PLUGIN_COMMAND is empty. Please set PLUGIN_COMMAND before running the script."
    exit 1
fi

# Initialize a variable to hold the constructed argument string
argument_string=""

# Iterate through the list of global options
for option in "${global_options[@]}"; do
    env_var_name="PLUGIN_LIQUIBASE_$(echo "$option" | tr '-' '_' | tr '[:lower:]' '[:upper:]')"

    # 3. If the resulting environment variable is set, append to the argument string
    if [ -n "${!env_var_name}" ]; then
        argument_string="$argument_string --$option ${!env_var_name}"
	# unset the environment variable to hide it from now on
	unset "$env_var_name"
    fi
done

argument_string="$argument_string $PLUGIN_COMMAND"


# Add all remaining PLUGIN_LIQUIBASE_ environment variables are command line params
for var in $(env | grep '^PLUGIN_LIQUIBASE_' | awk -F= '{print $1}'); do
    # Remove "PLUGIN_LIQUIBASE_" from the variable name, convert to lowercase, and replace underscores with hyphens
    var_name="${var#PLUGIN_LIQUIBASE_}"
    var_name_lower="$(echo "$var_name" | tr '[:upper:]' '[:lower:]' | tr '_' '-')"

    # Construct the string in the desired format
    argument_string="$argument_string --$var_name_lower ${!var}"
done


# Print the constructed argument string
command=`echo "/liquibase/liquibase $argument_string"`
echo "$command"

# Create unique file to avoid override in parallel steps
logfile=$(mktemp)

{ $command; } 2>&1 | tee -a "$logfile"

exit_code=${PIPESTATUS[0]}

encoded_command_logs=$(cat "$logfile" | base64 -w 0)

encoded_command_logs=`echo $encoded_command_logs | tr = -`

echo "encoded_command_logs=$encoded_command_logs" > "$DRONE_OUTPUT"
echo "exit_code=$exit_code" >> "$DRONE_OUTPUT"

