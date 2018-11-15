
.PHONY: build push

build:
	dep ensure
	go build

HUB?=
TAG?=
docker:
	sudo docker build  -t $(HUB)/url-lookup:$(TAG) .

push:
	sudo docker push $(HUB)/url-lookup:$(TAG)

