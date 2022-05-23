ECR_URL := 193635214029.dkr.ecr.eu-central-1.amazonaws.com/raid
ARGS :=

all: init run

init:
	go get

generate:
	go generate ./raid/

run:
	go run ./cmd/raid/main.go ${ARGS}

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run -E wsl,wrapcheck,nlreturn,revive,noctx,gocritic,errorlint,forcetypeassert

build-docker:
	docker build -t $(ECR_URL) .

run-docker: build-docker
	docker-compose up

push-docker: generate build-docker
	`AWS_PROFILE=adunai aws ecr get-login --no-include-email`
	docker push $(ECR_URL)
	docker logout

.PHONY: snapshots
snapshots:
	go run cmd/snapshots/main.go

.PHONY: video
video:
	mencoder "mf://snapshots/*.png" -o video.mp4 -ovc lavc -lavcopts vcodec=mjpeg -fps 60
