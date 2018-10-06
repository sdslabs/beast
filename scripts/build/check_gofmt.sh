#!/bin/bash

# Check the errors in formatting using gofmt
# Check formatting on non Godep'd code.
GOFMT_PATHS=$(find . -not -wholename "*.git*" -not -wholename "*Godeps*" -not -wholename "*vendor*" -not -name "." -type d)

# Find any files with gofmt problems
BAD_FILES=$(gofmt -s -l $GOFMT_PATHS)

if [ -n "$BAD_FILES" ]; then
  echo "The following files are not properly formatted:"
  echo $BAD_FILES
  exit 1
fi
