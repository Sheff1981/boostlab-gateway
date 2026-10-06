FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/boostlab-gateway ./cmd/gateway

FROM alpine:3.22
RUN adduser -D -H -s /sbin/nologin boostlab
COPY --from=build /out/boostlab-gateway /usr/local/bin/boostlab-gateway
USER boostlab
EXPOSE 8080/tcp 51821/udp
ENTRYPOINT ["/usr/local/bin/boostlab-gateway"]
