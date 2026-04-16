# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X buildinfo.Version=${VERSION} -X buildinfo.Build=${COMMIT} -X buildinfo.Date=${DATE}" \
    -o munin ./cmd/server

# Runtime stage
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/munin /munin

ENTRYPOINT ["/munin"]
