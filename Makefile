.PHONY: api assets build docker

assets:
	cd ./ui && yarn install --frozen-lockfile
	cd ./ui && yarn build

# This target skips the overhead of building UI assets.
# Intended to be used during development.
api:
	go build -o api ./cmd/asynqmon

# Build a release binary.
build: assets
	go build -o asynqmon ./cmd/asynqmon

# Build the image and run the Asynqmon server locally.
docker:
	docker build -t asynqmon .
	docker run --rm \
		--name asynqmon \
		-p 4000:8080 \
		--env AUTH_USERNAME=admin \
		--env AUTH_PASSWORD=admin \
		asynqmon --redis-addr=host.docker.internal:6379
