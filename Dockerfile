##
## Build
##
FROM golang:1.18-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY *.go ./

RUN CGO_ENABLED=0 go build -o /healthd

##
## Deploy
##
FROM alpine/curl
WORKDIR /

COPY --from=build /healthd /healthd

EXPOSE 8202

USER nobody:nobody

ENTRYPOINT ["/healthd", "-httpget", "http://localhost", "-filedontexists", "/tmp/healthd.flag"]