#!/bin/bash

URL="http://localhost:5005/auth/register"

echo "---- Testing Allowed Domain ----"
response_allowed=$(curl -s -o /dev/null -w "%{http_code}" -X POST $URL \
  -F "name=Neptune" \
  -F "username=neptune" \
  -F "password=romangod123" \
  -F "email=neptune@example.com")

if [ "$response_allowed" -eq 200 ]; then
  echo "✅ Test Passed-Allowed Domain"
else
  echo "❌ Test Failed-Allowed Domain (Got HTTP $response_allowed)"
fi

echo
echo "---- Testing Disallowed Domain ----"
response_disallowed=$(curl -s -o /dev/null -w "%{http_code}" -X POST $URL \
  -F "name=Poseidon" \
  -F "username=poseidon" \
  -F "password=greekgodsbetter" \
  -F "email=poseidon@gmail.com")

if [ "$response_disallowed" -eq 403 ]; then
  echo "✅ Test Passed - Disallowed Domain"
else
  echo "❌ Test Failed - Disallowed Domain (Got HTTP $response_disallowed)"
fi
