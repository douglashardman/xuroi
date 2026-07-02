.PHONY: dev infra api web seed workers test

dev:
	@chmod +x scripts/dev.sh
	@./scripts/dev.sh

infra:
	docker compose -f infra/docker-compose.yml up -d

api:
	cd api && go run ./cmd/xuroi

web:
	cd web && npm run dev

seed:
	cd api && go run ./cmd/seed

workers:
	@echo "Run in separate terminals:"
	@echo "  make -C api run-notify"
	@echo "  make -C api run-searchindex"
	@echo "  make -C api run-intelligence"

test:
	cd api && go test ./...