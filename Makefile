proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go-grpc_out=. proto/*.proto

build:
	@echo "Compiling nostradamus-api..."
	@mkdir -p bin
	@go build -o bin/api ./cmd/api

run:
	@echo "Running nostradamus-api..."
	@go run ./cmd/api

watch:
	@mkdir -p bin
	@CompileDaemon -color -build="go build -o bin/api ./cmd/api" -command="./bin/api" -log-prefix=false