ECR_URL := 193635214029.dkr.ecr.eu-central-1.amazonaws.com/raid

all: init run

init:
	go get

generate:
	go generate .

run:
	go run .

build-docker:
	docker build -t $(ECR_URL) .

run-docker: build-docker
	docker-compose up

push-docker: build-docker
	`AWS_PROFILE=adunai aws ecr get-login --no-include-email`
	docker push $(ECR_URL)
	docker logout
