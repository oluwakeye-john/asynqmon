.PHONY: api assets build docker docker-publish

DOCKER_IMAGE := oluwakeye/asynqmon
DOCKER_PLATFORMS ?= linux/amd64,linux/arm64

ifeq ($(firstword $(MAKECMDGOALS)),docker-publish)
DOCKER_VERSION := $(word 2,$(MAKECMDGOALS))
ifneq ($(word 3,$(MAKECMDGOALS)),)
$(error Usage: make docker-publish v0.2.0)
endif
ifneq ($(DOCKER_VERSION),)
.PHONY: $(DOCKER_VERSION)
$(DOCKER_VERSION):
	@:
endif
endif

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

# Build and publish a versioned multi-platform image plus the latest tag.
docker-publish:
	@test -n "$(DOCKER_VERSION)" || \
		{ echo "Usage: make docker-publish v0.2.0"; exit 1; }
	docker buildx build \
		--platform "$(DOCKER_PLATFORMS)" \
		--tag "$(DOCKER_IMAGE):$(DOCKER_VERSION)" \
		--tag "$(DOCKER_IMAGE):latest" \
		--push \
		.
