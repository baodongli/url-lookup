
.PHONY: build push

build:
	dep ensure
	go build

docker:
	sudo docker build  -t $HUB/url-lookup:$TAG .

push:
	sudo docker push $HUB/url-lookup:$TAG

