.PHONY: infra infra-down run migrate-up migrate-down

infra:
	docker-compose up -d

infra-down:
	docker-compose down

run:
	go run ./cmd/server/

migrate-up:
	go run ./cmd/migrate/ up

migrate-down:
	go run ./cmd/migrate/ down