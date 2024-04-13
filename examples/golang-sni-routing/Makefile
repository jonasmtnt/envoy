run:
	docker compose -f docker-compose-go.yaml run --rm go_plugin_compile
	docker compose pull
	docker compose up --build -d
	docker compose ps

test:
	psql -h localhost -p 5432 -U postgres