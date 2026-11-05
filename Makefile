.PHONY: run test fmt compose

run:
	go run ./cmd/web

test:
	go test ./...

fmt:
	gofmt -w .

compose:
	docker compose up --build
