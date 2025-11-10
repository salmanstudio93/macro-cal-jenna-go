#!/bin/bash

# Test script for serving selection API
echo "Testing serving selection API..."

# Test the serving selection endpoint
echo "Sending request to /serving-selection endpoint..."
curl -X POST http://localhost:8080/serving-selection \
  -H "Content-Type: application/json" \
  -d @test_serving_selection.json \
  -w "\nHTTP Status: %{http_code}\n" \
  -s

echo ""
echo "Test completed!"
