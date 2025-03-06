
# build go app
FROM golang:1.24-alpine AS build
RUN mkdir /build
ADD . /build
WORKDIR /build
RUN go build -o garden-app .

FROM alpine:latest AS demo
RUN mkdir /app
WORKDIR /app
COPY --from=build /build/garden-app .
COPY gardens.yaml.example /app/gardens.yaml
COPY config.yaml.example /app/config.yaml
ENTRYPOINT ["/app/garden-app", "serve", "--readonly"]

FROM alpine:latest AS production
RUN mkdir /app
WORKDIR /app
COPY --from=build /build/garden-app .
ENTRYPOINT ["/app/garden-app"]
