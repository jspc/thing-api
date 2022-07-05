VERSION ?= "latest"
IMG ?= jspc/thing-api:$(VERSION)

default: thing-api

docs/docs.go docs/swager.json docs/swagger.yaml: main.go api.go
	swag init

thing-api: docs/docs.go *.go
	CGO_ENABLED=0 go build -o app -ldflags="-s -w" && upx app

.PHONY: docker docker-push
docker: docs/docs.go
	docker build -t $(IMG) .

docker-push:
	docker push $(IMG)
