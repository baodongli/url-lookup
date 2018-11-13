
.PHONY: build push

build:
	dep ensure
	go build
