GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
BINARY_NAME=mongo-bench
MAIN_FILE=main.go

.PHONY: all
all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BINARY_NAME)"

run:
	@echo "Running $(BINARY_NAME) with THREADS=$(THREADS), DOCS=$(DOCS), URI=$(URI)"
	./$(BINARY_NAME) -threads $(THREADS) -docs $(DOCS) -uri $(URI)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	@echo "Clean complete"

test:
	$(GOTEST) -v ./...

help:
	@echo "Usage:"
	@echo "  make          - Builds the binary"
	@echo "  make build    - Builds the binary"
	@echo "  make run      - Runs the application with THREADS, DOCS, and URI variables"
	@echo "                 Example: make run THREADS=10 DOCS=10000 URI=mongodb://localhost:27017"
	@echo "  make clean    - Removes the binary"
	@echo "  make test     - Runs tests"
	@echo "  make help     - Displays this help message"