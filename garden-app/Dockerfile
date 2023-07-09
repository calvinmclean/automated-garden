# build static svelte app
FROM node:20-alpine as svelte_build
RUN mkdir /build
ADD ./frontend /build
ENV NODE_ENV=docker
WORKDIR /build
RUN npm install && npm run build

# build go app
FROM golang:1.19-alpine AS build
RUN mkdir /build
ADD . /build
WORKDIR /build
COPY --from=svelte_build /build/dist ./server/dist
RUN go build -o garden-app .

FROM alpine:latest AS production
RUN mkdir /app
WORKDIR /app
COPY --from=build /build/garden-app .
ENTRYPOINT ["/app/garden-app"]
