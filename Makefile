.PHONY: up down build logs migrate-create

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose up -d --build

logs:
	docker compose logs -f

migrate-create:
	@read -p "Nome da migracao (ex: init_schema): " name; \
	docker run --rm -v $(PWD)/backend/migrations:/migrations migrate/migrate create -ext sql -dir /migrations -seq $$name
