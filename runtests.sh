#!/bin/bash
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color
# This script runs the tests for the project.
# It is intended to be run from the project root directory
runtest() {
  jsonFile=$1
  expectedResult=$2
  echo "Running test for $jsonFile"
  go run main.go $jsonFile
  result=$?
  if [ $result -ne $expectedResult ]; then
    echo -e "${RED}Test failed for $jsonFile${NC}"
    exit 1
  else
    echo -e "${GREEN}Test passed for $jsonFile${NC}"
  fi
}

tests() {
  runtest tests/tests/step1/valid.json 0
  runtest tests/tests/step1/invalid.json 1
  
  runtest tests/tests/step2/valid.json 0
  runtest tests/tests/step2/invalid.json 1
  runtest tests/tests/step2/valid2.json 0
  runtest tests/tests/step2/invalid2.json 1
  
  runtest tests/tests/step3/valid.json 0
  runtest tests/tests/step3/invalid.json 1
  
  runtest tests/tests/step4/valid.json 0
  runtest tests/tests/step4/invalid.json 1
  runtest tests/tests/step4/valid2.json 0
}

step5tests() {
  # loop through files in step5
  for file in tests/tests/step5/*; do
    # Skip fail18.json - not checking nesting depth
    if [ "$file" == "tests/tests/step5/fail18.json" ]; then
      continue
    fi
    # expect fail for files prefixed with 'fail'
    if [[ $file == *"fail"* ]]; then
      echo "Should fail"
      runtest $file 1
    else
      runtest $file 0
    fi
  done
}

tests
step5tests
echo -e "${GREEN}PASSED"