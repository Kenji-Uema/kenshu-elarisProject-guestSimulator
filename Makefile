IMAGE_TAG ?= 1.0.0

build: generate
	go build .

generate:
	buf generate

docker-build:
	docker buildx build --build-arg SERVICE_NAME=guest-simulator --build-arg VERSION=$(IMAGE_TAG) -t guest-simulator:$(IMAGE_TAG) --load .
