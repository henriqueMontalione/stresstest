BINARY=stresstest
IMAGE=stresstest

.PHONY: build run test lint docker-build docker-run clean

build:
	go build -o $(BINARY) ./cmd/stresstest

run:
	go run ./cmd/stresstest $(ARGS)

test:
	go test ./... -count=1

lint:
	go vet ./...
	go build ./...

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run $(IMAGE) $(ARGS)

clean:
	rm -f $(BINARY)
