#!/bin/bash

# Load environment variables from .env file
# Using a while loop to properly handle special characters
while IFS='=' read -r key value; do
  # Skip comments and empty lines
  if [[ ! "$key" =~ ^# && -n "$key" ]]; then
    # Remove leading/trailing whitespace
    key=$(echo "$key" | xargs)
    value=$(echo "$value" | xargs)
    export "$key=$value"
  fi
done < .env

# Run the Go program
go run .
