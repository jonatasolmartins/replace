build:
	@go build -o bin/replace

run: build
	@./bin/replace

test:
	@go test -v ./testing/unit_test.go

