IMAGE_TAG ?= 2.0.5

build: generate
	go build .

generate:
	npx buf generate

docker-build:
	docker buildx build --build-arg SERVICE_NAME=guest-simulator --build-arg VERSION=$(IMAGE_TAG) -t guest-simulator:$(IMAGE_TAG) --load .
