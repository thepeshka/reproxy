FROM golang:1.26 AS build
COPY . /app
WORKDIR /app
RUN go mod download && \
    go mod verify && \
    go build -ldflags '-linkmode external -extldflags "-fno-PIC -static"' -v -o /usr/bin

FROM alpine:3.19

COPY --from=build /usr/bin/reporxy /usr/bin/reporxy
ENTRYPOINT ["/usr/bin/reporxy"]
