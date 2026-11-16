.PHONY: run test fmt compose check-links docs-check

run:
	go run ./cmd/web

test:
	go test ./...

fmt:
	gofmt -w .

compose:
	docker compose up --build

# Нужен интернет: проверяет, что ссылки на материалы в схемах roadmap открываются.
check-links:
	CHECK_LINKS=1 go test ./internal/content -run TestLinksAlive -v

# Проверяет документацию по стандарту docs/standards/documentation-standard.md.
docs-check:
	go test ./internal/docscheck -v
