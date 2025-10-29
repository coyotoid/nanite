.PHONY: build install
PREFIX=$(HOME)/.local

build:
	go build .

install: build
	install -Dd $(PREFIX)/bin
	install nanite $(PREFIX)/bin/nanite
