# MongoDB Benchmarking Tool - mongo-bench

mongo-bench is a high-performance benchmarking tool written in Golang designed to test and measure MongoDB insert and 
update rates under configurable conditions. This tool is useful for database engineers, developers, 
and system administrators who want to assess MongoDB performance by simulating multiple 
threads performing insert or update operations in a MongoDB collection.

## Features

- Configurable Load Testing: Set the number of concurrent threads and the total number of documents to insert or update.
- Insert and Update Modes:
- Insert Mode: Inserts documents into the MongoDB collection.
- Update Mode: Updates previously inserted documents, simulating real-world workloads with mixed read-write operations.
- High-Resolution Metrics: Captures and logs operation rates every second, including:
- Total document count
- Mean operation rate
- m1_rate, m5_rate, m15_rate (1-minute, 5-minute, and 15-minute moving average rates)
- Mean rate (mean_rate)
- In-Memory Logging with Final CSV Export: Stores per-second metrics in memory and exports to a CSV file after the test completes, minimizing disk I/O during the benchmark run.
- Detailed Console Output: Logs real-time performance metrics to stdout every second.

## Usage

After building the tool, run it with customizable parameters:
```bash
./mongo-bench -threads <number_of_threads> -docs <number_of_documents> -uri <mongodb_uri> -type <test_type>
```

Parameters

- -threads: Number of concurrent threads to use for inserting or updating documents.
- -docs: Total number of documents to insert or update during the benchmark.
- -uri: MongoDB connection URI.
- -type: Type of test to run. Accepts either insert or update:
- insert: The tool will insert new documents.
- update: The tool will update existing documents (requires that documents have been inserted in a prior run).

Example Commands

### Insert Test:

```bash
./mongo-bench -threads 10 -docs 100000 -uri mongodb://localhost:27017 -type insert
```

This will insert 100,000 documents into MongoDB using 10 concurrent threads.

### Update Test:

```bash
./mongo-bench -threads 10 -docs 100000 -uri mongodb://localhost:27017 -type update
```

This will update the 100,000 documents previously inserted using 10 concurrent threads.

Output

- Console: Logs per-second operation rate metrics to stdout.
- CSV: After the test completes, saves a detailed CSV file (benchmark_results.csv) with columns:
- t: Timestamp (epoch seconds)
- count: Total document count
- mean: Mean operation rate in docs/sec
- m1_rate, m5_rate, m15_rate: Moving average rates over 1, 5, and 15 minutes, respectively
- mean_rate: Cumulative mean rate

This CSV file provides an in-depth view of performance over time, which can be used for analysis or visualizations.

Example CSV Output
```text
t,count,mean,m1_rate,m5_rate,m15_rate,mean_rate
1730906793,100000,30000.50,31000.12,30500.45,30000.25,29900.88
```

## Building the Tool

Build the tool using the provided Makefile:
```bash
make build
```

## License

This project is licensed under the MIT License.

This README includes details for both the insert and update functionalities, 
along with usage instructions for each mode. Let me know if you need further customization!