.PHONY: build build-web deploy restart stop status dev-web dev-api

export PATH := /usr/local/go/bin:$(PATH)

build-web:
	cd web && npm ci && npm run build

build: build-web
	go build -o home-router .

dev-web:
	cd web && npm run dev

dev-api:
	go run .

deploy: build
	sudo systemctl stop home-router || true
	sudo cp home-router /usr/local/bin/home-router
	sudo cp config.yml /etc/home-router/config.yml
	sudo systemctl start home-router

restart:
	sudo systemctl restart home-router

stop:
	sudo systemctl stop home-router

status:
	sudo systemctl status home-router
