ECR_URL := 193635214029.dkr.ecr.eu-central-1.amazonaws.com/raid

build:
	docker build -t $(ECR_URL) .

push: build
	`AWS_PROFILE=adunai aws ecr get-login --no-include-email`
	docker push $(ECR_URL)
	docker logout
