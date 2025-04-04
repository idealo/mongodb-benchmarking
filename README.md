
# MongoDB Benchmarking Tool - mongo-bench

![CodeQL](https://github.com/idealo/mongodb-benchmarking/actions/workflows/codeql.yml/badge.svg)

`mongo-bench` is a high-performance benchmarking tool written in Golang designed to test and measure 
MongoDB insert, update, delete, and upsert rates under configurable conditions. 
This tool is useful for database engineers, developers, and system administrators who want to assess MongoDB 
performance by simulating multiple threads performing operations in a MongoDB collection.

## Features

- **Configurable Load Testing**: Set the number of concurrent threads and the total number of documents for testing.
- **Modes of Operation**:
  - **Insert Mode**: Inserts documents into the MongoDB collection.
  - **Update Mode**: Updates previously inserted documents, simulating real-world workloads with mixed read-write operations.
  - **Delete Mode**: Deletes existing documents from the MongoDB collection.
  - **Upsert Mode**: Performs upserts on documents, ensuring repeated upserts within a specified range.
  - **Run-All Sequence**: Runs the insert, update, delete, and upsert tests in sequence, providing a comprehensive performance assessment.
- **High-Resolution Metrics**: Captures and logs operation rates every second, including:
  - Total document count
  - Mean operation rate
  - m1_rate, m5_rate, m15_rate (1-minute, 5-minute, and 15-minute moving average rates)
  - Mean rate (mean_rate)
- **In-Memory Logging with Final CSV Export**: Stores per-second metrics in memory and exports to a CSV file after the test completes, minimizing disk I/O during the benchmark run.
- **Detailed Console Output**: Logs real-time performance metrics to stdout every second.

## Usage

After building the tool, run it with customizable parameters:

```bash
./mongo-bench -threads <number_of_threads> -docs <number_of_documents> -uri <mongodb_uri> -type <test_type>
```

### Parameters

- `-threads`: Number of concurrent threads to use for inserting, updating, deleting, or upserting documents.
- `-docs`: Total number of documents to process during the benchmark.
- `-duration`: Duration of the test in seconds (default: 0 seconds).
- `-largeDocs`: Use large documents (2K) (default: false).
- `-dropDb`: Drop the database before running the test (default: true).
- `-uri`: MongoDB connection URI.
- `-useIndex`: Create indexes (just in `insertDoc` mode), use them in `findDoc`
- `-out`: Output file prefix. (Default: empty, using "benchmark_results_*")

- `-type`: Type of test to run. Accepts `insert`, `update`, `delete`, `upsert`, `insertDoc`, `findDoc`, `runDoc` or `runAll`:
  - `insert`: The tool will insert new documents.
  - `insertDoc`: The tool will insert new more complex documents to simulate real-life data.
  - `findDoc`: The tool will create random find requests on the complex documents created by `insertDoc` operation to simulate real-life requests.
  - `update`: The tool will update existing documents (requires that documents have been inserted in a prior run).
  - `delete`: The tool will delete existing documents. (just if `docs` is given)
  - `upsert`: The tool will perform upserts, repeatedly updating a specified range. (just if `docs` is given)
  - `runAll`: Runs the `insert`, `update`, `delete`, and `upsert` tests sequentially. (just if `docs` is given)
  - `runAll`: Runs the `insert`, `insertDoc`, `update` tests sequentially. (just if `duration` is given)
  - `runDoc`: Runs the `insertDoc` and `findDoc` tests sequentially.

### Example Commands

#### Insert Test:

```bash
./mongo-bench -threads 10 -docs 100000 -uri mongodb://localhost:27017 -type insert
```

This command will insert 100,000 documents into MongoDB using 10 concurrent threads.

#### Update Test:

```bash
./mongo-bench -threads 10 -docs 100000 -uri mongodb://localhost:27017 -type update
```

This command will update the 100,000 documents previously inserted using 10 concurrent threads.

#### Delete Test:

```bash
./mongo-bench -threads 10 -docs 100000 -uri mongodb://localhost:27017 -type delete
```

This command will delete documents from MongoDB using 10 concurrent threads.

#### Upsert Test:

```bash
./mongo-bench -threads 10 -docs 100000 -uri mongodb://localhost:27017 -type upsert
```

This command will perform upserts on documents within a specified range, using 10 concurrent threads.

#### Run All Tests:

```bash
./mongo-bench -threads 10 -docs 100000 -uri mongodb://localhost:27017 --runAll
```

This command will run the `insert`, `update`, `delete`, and `upsert` tests sequentially using 10 concurrent threads.

## Output

- **Console**: Logs per-second operation rate metrics to stdout.
- **CSV**: After the test completes, saves a detailed CSV file (e.g., `benchmark_results_insert.csv`, `benchmark_results_update.csv`) with columns:
  - `t`: Timestamp (epoch seconds)
  - `count`: Total document count
  - `mean`: Mean operation rate in docs/sec
  - `m1_rate`, `m5_rate`, `m15_rate`: Moving average rates over 1, 5, and 15 minutes, respectively
  - `mean_rate`: Cumulative mean rate

This CSV file provides an in-depth view of performance over time, which can be used for analysis or visualizations.

### Example CSV Output
```text
t,count,mean,m1_rate,m5_rate,m15_rate,mean_rate
1730906793,100000,30000.50,31000.12,30500.45,30000.25,29900.88
```

## Building the Tool

Build the tool using the provided Makefile:

```bash
make build
```


## Document Generator

In addition to basic synthetic documents, the benchmark suite includes a **document generator** that produces structured, content-rich documents. These are designed to simulate more realistic application scenarios, such as articles in a CMS or entries in a blog or knowledge base.

The generated documents follow a fixed structure and are suitable for testing more complex operations such as filtering, range queries, and full-text search. Each document includes both metadata and content fields, aiming to reflect common patterns in document-oriented data models.

**Document structure:**
- `_id`: autogenerated `ObjectID`
- `threadRunCount`: internal thread identifier (for debugging or tracing)
- `rnd`: a random number, e.g. for versioning
- `v`: static version field
- `title`: short title generated from random text
- `author`: a randomly chosen author name
- `co_authors`: list of 1–3 additional authors
- `summary`: short summary text
- `content`: longer body text (2,000–5,000 characters, randomized)
- `tags`: list of 4–6 tags from a predefined set
- `category`: one of several topic categories
- `timestamp`: random date from the last two years
- `views`, `comments`, `likes`, `shares`: numeric counters for simulated engagement

This generator is used in the `"insertDoc"` and `"findDoc"` operations and enables benchmarking scenarios that are closer to real-world usage patterns while maintaining reproducibility.

## Query Generator

To support realistic **read benchmarks**, the suite includes a `QueryGenerator` used in the `findDoc` operation. Instead of issuing uniform or synthetic queries, this generator produces a variety of randomized but meaningful MongoDB filters that reflect real-world use cases.

Each query is randomly selected from a set of common patterns:

- **Author match**: Finds documents by a specific author.
- **Tag match**: Selects documents containing a specific tag using `$elemMatch`.
- **Date range**: Retrieves documents newer than a randomly chosen date within the past year.
- **Full-text search**: Executes `$text` search over indexed content fields (requires a text index).

This variety ensures that find benchmarks test different aspects of database performance, such as indexing, full-text search, and filter efficiency. The generator runs per thread and per query, introducing realistic and randomized load patterns.


## License

This project is licensed under the MIT License.
