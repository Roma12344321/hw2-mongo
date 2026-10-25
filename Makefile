.PHONY: help build test lint docker-build up down clean fixtures reset reset-index data index purchase failover all

LAB := docker compose run --rm app

help:
	@go run ./cmd/lab help

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

# Пересборка образа приложения после правок кода.
docker-build:
	docker compose build app

up:
	docker compose up -d --wait

down:
	docker compose down

clean:
	docker compose down -v

fixtures: ## Показать контрольные данные и эталоны
	$(LAB) fixtures

reset: ## Восстановить 6 товаров и 35 заказов
	$(LAB) reset

reset-index: ## Загрузить набор из 10 000 заказов
	$(LAB) reset index

data: ## Валидатор, поиск, история, отчёт
	$(LAB) data

index: ## Планы до/после составного индекса
	$(LAB) index

purchase: ## Покупка: успех, откат, повтор, конфликт
	$(LAB) purchase

failover: ## Смена primary тем же клиентом
	$(LAB) failover

all: ## Полный прогон четырёх сценариев
	$(LAB) all
