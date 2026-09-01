include .env
export

export PROJECT_ROOT=${shell pwd}

env-up: 
	@docker compose up -d postgres redis

env-down: 
	@docker compose down postgres redis

env-cleanup: 
	@read -p "Очистить все volume файлы окружения? Опасность утери данных. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down postgres redis port-forwarder && \
		rm -rf ${PROJECT_ROOT}/out/pgdata && \
		rm -rf ${PROJECT_ROOT}/out/redis_data && \
		echo "Файлы окружения очищены"; \
	else \
		echo "Очистка окружения отменена"; \
	fi

env-port-forward:
	@docker compose up -d port-forwarder


env-port-close:
	@docker compose down port-forwarder


migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "Отсутствует необходимый параметр 'seq'. Пример: make migrate-create seq=init"; \
		exit 1; \
	fi; \
	docker compose run --rm postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"  && \
	sudo chown $$(id -u):$$(id -g) ${PROJECT_ROOT}/migrations/*.sql && \
	echo "✅ Миграция создана и права скорректированы"

	

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Отсутствует необходимый параметр 'action'. Пример: make migrate-action action=up"; \
		exit 1; \
	fi; \
	docker compose run --rm postgres-migrate \
		-path /migrations \
		-database postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable \
		"$(action)"

migrate-up:
	@make migrate-action action=up


migrate-down:
	@make migrate-action action=down


logs-cleanup:
	@read -p "Очистить все log файлы окружения? Опасность утери логов. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		sudo rm -rf ${PROJECT_ROOT}/out/logs && \
		echo "Файлы логов очищены"; \
	else \
		echo "Очистка логов отменена"; \
	fi;


ps:
	@docker compose ps

tcp-deploy:
	@docker compose up -d --build tcp

tcp-undeploy:
	@docker compose down tcp

run:
	@export LOGGER_FOLDER=${PROJECT_ROOT}/out/logs && \
	export POSTGRES_HOST=localhost && \
	sudo chmod -R 777 ${PROJECT_ROOT}/out/pgdata && \
	go mod tidy && \
	go run cmd/tcp/main.go