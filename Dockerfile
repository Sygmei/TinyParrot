# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-s -w -buildid=" \
    -o /out/tinyparrot .

FROM scratch
COPY --from=build /out/tinyparrot /tinyparrot
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/tinyparrot"]
