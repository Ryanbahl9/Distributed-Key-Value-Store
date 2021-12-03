# syntax=docker/dockerfile:1

FROM golang:latest

WORKDIR /webserver

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

ENV GIN_MODE=release

RUN go build -o /gowebserver

EXPOSE 8090

CMD [ "/gowebserver" ]