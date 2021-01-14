.DEFAULT_GOAL := run

version = `git fetch --tags >/dev/null && git describe --tags | cut -c 2-`
docker_container = akarasz/dajsz-slack
docker_tags = $(version),latest

.PHONY := build
build:
	go build ./...

.PHONY := test
test:
	go test ./...

.PHONY := docker
docker:
	docker build -t "$(docker_container):latest" -t "$(docker_container):$(version)" .

.PHONY := run
run:
	go run cmd/app/main.go

push: docker
	docker push $(docker_container):latest
	docker push $(docker_container):$(version)
