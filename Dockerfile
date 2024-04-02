FROM golang:1.22-alpine3.19 AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o external-dns-abion -ldflags '-w -extldflags "-static"' .

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --from=build /workspace/external-dns-abion /usr/local/bin/external-dns-abion

USER 20000:20000

ENTRYPOINT ["external-dns-abion"]