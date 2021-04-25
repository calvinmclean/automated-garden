FROM golang:1.16-alpine AS build
RUN mkdir /build
ADD . /build
WORKDIR /build
RUN go build -o garden-app .

FROM alpine:latest AS production
RUN mkdir /app
WORKDIR /app
COPY --from=build /build/garden-app .
ENTRYPOINT ["/app/garden-app"]
