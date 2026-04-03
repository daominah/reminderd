#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status

# run this script in project root directory before commit

# optional format the whole repo
echo "----------------------------------------"
echo "running goimports ..."
goimports -w .
echo "end goimports"

# Go Vet examines Go source code and reports suspicious constructs
echo "----------------------------------------"
echo "running go vet ..."
go vet ./...
echo "end go vet"

# Go Fix updates code to modern Go APIs and patterns.
# Show what go fix would change (does not apply changes)
echo "----------------------------------------"
echo "running go fix -diff ..."
go fix -diff ./...
echo "end go fix -diff"

# StaticCheck is deeper than Go Vet, analysis for bugs, performance issues, deprecated APIs.
# To install/update the tool: go install honnef.co/go/tools/cmd/staticcheck@latest
echo "----------------------------------------"
echo "running staticcheck ..."
staticcheck ./...
echo "end staticcheck"

# Run all unittests, include database tests, which require Docker compose to be running.
# go clean -testcache
echo "----------------------------------------"
echo "running go test ..."
go test -v ./...
# go test ./...
echo "end go test"

echo "========================================"
echo "All tests passed!"
