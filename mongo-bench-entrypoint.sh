#!/bin/bash

# Wait for MongoDB to be ready
until nc -z mongodb 27017; do
  echo 'Waiting for MongoDB...'
  sleep 2
done

# Run the insert test
echo 'Running insert test...'
./mongo-bench --uri mongodb://root:example@mongodb:27017 --type insert --threads 11 --docs 80055

# Check the document count and verify it's 1000
echo 'Checking document count...'
DOC_COUNT=$(mongosh 'mongodb://root:example@mongodb:27017/?authSource=admin' --quiet --eval 'JSON.stringify({count: db.getSiblingDB("benchmarking").testdata.countDocuments()})' | jq -r '.count')

if [ -z "$DOC_COUNT" ]; then
  echo 'Error: Failed to retrieve document count.'
  exit 1
elif [ "$DOC_COUNT" -ne 80055 ]; then
  echo "Error: Expected 80055 documents, found $DOC_COUNT"
  exit 1
fi

# Run the update test
echo 'Running update test...'
./mongo-bench --uri mongodb://root:example@mongodb:27017 --type update --threads 10 --docs 80055
echo 'Checking document count...'
DOC_COUNT=$(mongosh 'mongodb://root:example@mongodb:27017/?authSource=admin' --quiet --eval 'JSON.stringify({count: db.getSiblingDB("benchmarking").testdata.countDocuments()})' | jq -r '.count')
if [ "$DOC_COUNT" -ne 80055 ]; then
  echo "Error: Expected 80055 documents, found $DOC_COUNT"
  exit 1
fi


# Run the delete test
echo 'Running delete test...'
./mongo-bench --uri mongodb://root:example@mongodb:27017 --type delete --threads 10 --docs 80055

echo 'Checking document count...'
DOC_COUNT=$(mongosh 'mongodb://root:example@mongodb:27017/?authSource=admin' --quiet --eval 'JSON.stringify({count: db.getSiblingDB("benchmarking").testdata.countDocuments()})' | jq -r '.count')
if [ "$DOC_COUNT" -ne 0 ]; then
  echo "Error: Expected 0 documents, found $DOC_COUNT"
  exit 1
fi

# Run the upsert test
echo 'Running upsert test...'
./mongo-bench --uri mongodb://root:example@mongodb:27017 --type upsert --threads 10 --docs 80000

echo 'Checking document count...'
DOC_COUNT=$(mongosh 'mongodb://root:example@mongodb:27017/?authSource=admin' --quiet --eval 'JSON.stringify({count: db.getSiblingDB("benchmarking").testdata.countDocuments()})' | jq -r '.count')
if [ "$DOC_COUNT" -gt 0 ]; then
  echo 'Single tests passed successfully.'
else
  echo "Error: Expected >0 documents, found $DOC_COUNT"
  exit 1
fi

# Run the all test
echo 'Running all test...'
./mongo-bench --uri mongodb://root:example@mongodb:27017 --runAll --threads 10 --docs 80000
if [ $? -ne 0 ]; then
  echo 'Error: docs test with runAll failed.'
  exit 1
fi

echo 'Running duration test...'
./mongo-bench --duration 10  --threads 10 -type insert --uri mongodb://root:example@mongodb:27017
if [ $? -ne 0 ]; then
  echo 'Error: duration test with insert failed.'
  exit 1
fi

./mongo-bench --duration 10  --threads 10 -type update --uri mongodb://root:example@mongodb:27017
if [ $? -ne 0 ]; then
  echo 'Error: duration test with update failed.'
  exit 1
fi