shell := bash

IMAGE_NAME = astma/crabby

ci:
	go get -u github.com/golang/dep/cmd/dep
	make dep
	make build

dep:
	DEPNOLOCK=1 dep ensure

build:
	go build ./

image:
	docker build -t $(IMAGE_NAME) .

push-image:
	docker push $(IMAGE_NAME)
