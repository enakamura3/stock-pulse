.PHONY: up down build logs migrate-create test-backend test-frontend e2e

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

test-backend:
	cd backend && go test -v ./...

test-frontend:
	docker compose run --rm frontend npm run test:coverage

e2e:
	./scripts/run-e2e.sh
