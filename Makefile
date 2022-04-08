ECR_URL := 193635214029.dkr.ecr.eu-central-1.amazonaws.com/raid

all: init run

init:
	go get

run:
	go generate .
	go run .

build:
	go generate .
	docker build -t $(ECR_URL) .

push: build
	`AWS_PROFILE=adunai aws ecr get-login --no-include-email`
	docker push $(ECR_URL)
	docker logout
