# If you need help implementing this file, check out the Dockerfile in the auth-service or the homework!
FROM golang:latest

ADD . /go/src/github.com/BearCloud/fa20-project-dev/profiles-service

WORKDIR /go/src/github.com/BearCloud/fa20-project-dev/profiles-service

RUN go mod download

RUN go build -o main .

EXPOSE 80

ENTRYPOINT [ "./main" ]