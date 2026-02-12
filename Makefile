build: generate
	go build .

generate:
	npx buf generate

docker-build:
	 docker build --build-arg SERVICE_NAME=guest-emulator --build-arg VERSION=latest -t guest-emulator:latest .