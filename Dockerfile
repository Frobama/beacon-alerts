# syntax=docker/dockerfile:1
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/beacon-alerts ./cmd/alerts

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/beacon-alerts /app/beacon-alerts
USER 65532:65532
EXPOSE 8082 9091
ENTRYPOINT ["/app/beacon-alerts"]
