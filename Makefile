shell := bash

IMAGE_NAME = astma/crabby

build:
	go build ./

image:
	docker build -t $(IMAGE_NAME) .

push-image:
	docker push $(IMAGE_NAME)
