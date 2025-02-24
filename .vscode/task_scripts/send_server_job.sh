#!/bin/bash
# This script sends a POST request to the server running on localhost:8082/submit
# Adjust the URL or payload as needed.

# Define the URL and JSON payload.
URL="http://localhost:8082/submit"
PAYLOAD='{
  "input_file_path": "input.mp4",
  "output_file_path": "output.mp4",
  "container_type": "mp4",
  "flags": "--dry-run"
}'

echo "Sending POST request to $URL..."
echo "Payload: $PAYLOAD"

# Send the POST request with curl.
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}\n" \
  -X POST \
  -H "Content-Type: application/json" \
  -d "${PAYLOAD}" \
  "$URL")

echo "Response from server:"
echo "$RESPONSE"