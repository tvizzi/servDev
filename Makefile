deploy:
	git pull origin main
	docker-compose down
	docker compose up --build -d

restart:
	docker-compose restart