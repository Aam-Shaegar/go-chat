include .env
export

.PHONY: infra infra-down migrate-up migrate-down migrate-create migrate-action run

infra:
	@docker compose up -d gochat-postgres port-forwarder redis

infra-down:
	@docker compose down

migrate-up:
	@make migrate-action action=up

migrate-down:
	@make migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Error: action is required. Usage: make migrate-action action=<action>"; \
		exit 1; \
	fi; \
		docker compose run --rm gochat-migrate \
			-path /migrations \
			-database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@gochat-postgres:5432/$(POSTGRES_NAME)?sslmode=disable" \
			"$(action)"

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "Error: migration name required. Usage: make migrate-create seq=<name>"; \
		exit 1; \
	fi; \
	docker compose run --rm --entrypoint="" gochat-migrate \
		migrate create -ext sql -dir /migrations -seq "$(seq)"

run:
	@export POSTGRES_HOST=localhost && \
		go run ./cmd/server/main.go

frontend-dev:
	@cd frontend && npm run dev

frontend-build:
	@cd frontend && npm run build
