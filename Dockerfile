# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" \
    -o /out/radius-director \
    ./cmd/radius-director

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/radius-director /app/radius-director
COPY templates /app/factory/templates
COPY schemas /app/factory/schemas

ENV RADIUS_DIRECTOR_ASSETS=/assets

ENTRYPOINT ["/app/radius-director"]