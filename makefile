run:
	@go run cmd/*.go

build:
	@go build -o ./bin/main cmd/*.go

docker-image:
	@docker build -t recorder-dev .