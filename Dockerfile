# syntax=docker/dockerfile:1.7

#
# First stage: 
# Building a frontend.
#

FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend

# Move to a working directory (/static).
WORKDIR /static

# Install dependencies separately so Docker can cache them until the lockfile changes.
COPY ui/package.json ui/yarn.lock ./
RUN --mount=type=cache,target=/usr/local/share/.cache/yarn \
    yarn install --frozen-lockfile

# Copy the UI source and create the production bundle.
COPY ui .
RUN yarn build

#
# Second stage: 
# Building a backend.
#

FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.24 AS backend

# Move to a working directory (/build).
WORKDIR /build

# Install the CA bundle used by TLS Redis connections in the scratch image.
RUN apk add --no-cache ca-certificates

# Copy and download dependencies.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy a source code to the container.
COPY . .

# Copy frontend static files from /static to the root folder of the backend container.
COPY --from=frontend ["/static/build", "ui/build"]

# Build for Docker's requested target platform rather than the builder's platform.
ARG TARGETOS
ARG TARGETARCH

# Run go build (with ldflags to reduce binary size).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o asynqmon ./cmd/asynqmon

#
# Third stage: 
# Creating and running a new scratch container with the backend binary.
#

FROM scratch

# Include system roots so verified TLS Redis connections work.
COPY --from=backend ["/etc/ssl/certs/ca-certificates.crt", "/etc/ssl/certs/ca-certificates.crt"]

# Copy binary from /build to the root folder of the scratch container.
COPY --from=backend ["/build/asynqmon", "/"]

# Run as an unprivileged numeric user; scratch has no user database.
USER 65532:65532

EXPOSE 8080

# Command to run when starting the container.
ENTRYPOINT ["/asynqmon"]
