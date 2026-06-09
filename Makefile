.PHONY: up down restart logs rebuild

# Запустить контейнеры (с пересборкой)
up:
	docker-compose up --build

# Запустить в фоне
up-d:
	docker-compose up -d --build

# Остановить и удалить контейнеры
down:
	docker-compose down

# Перезапустить (down + up)
restart: down up

# Посмотреть логи
logs:
	docker-compose logs -f

# Пересобрать без кэша и запустить
rebuild:
	docker-compose down
	docker-compose build --no-cache
	docker-compose up