#!/bin/bash

# Wait for MongoDB to be ready
until nc -z mongodb 27017; do
  echo 'Waiting for MongoDB...'
  sleep 2
done

# Run the insert test
echo 'Running insert test...'
./mongo-bench --uri mongodb://root:example@mongodb:27017 --type insert --threads 10 --docs 80000

# Check the document count and verify it's 1000
echo 'Checking document count...'
DOC_COUNT=$(mongosh 'mongodb://root:example@mongodb:27017/?authSource=admin' --quiet --eval 'JSON.stringify({count: db.getSiblingDB("benchmarking").testdata.countDocuments()})' | jq -r '.count')

if [ -z "$DOC_COUNT" ]; then
  echo 'Error: Failed to retrieve document count.'
  exit 1
elif [ "$DOC_COUNT" -ne 80000 ]; then
  echo "Error: Expected 80000 documents, found $DOC_COUNT"
  exit 1
fi

# Run the delete test
echo 'Running delete test...'
./mongo-bench --uri mongodb://root:example@mongodb:27017 --type delete --threads 10 --docs 80000

echo 'Checking document count...'
DOC_COUNT=$(mongosh 'mongodb://root:example@mongodb:27017/?authSource=admin' --quiet --eval 'JSON.stringify({count: db.getSiblingDB("benchmarking").testdata.countDocuments()})' | jq -r '.count')
if [ "$DOC_COUNT" -ne 0 ]; then
  echo "Error: Expected 0 documents, found $DOC_COUNT"
  exit 1
else
  echo 'All tests passed successfully.'
fi