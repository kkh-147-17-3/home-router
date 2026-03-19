.PHONY: build deploy restart stop status

export PATH := /usr/local/go/bin:$(PATH)

build:
	go build -o home-router .

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
