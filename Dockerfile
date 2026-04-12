FROM golang:1.26.2 AS build
WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/go-logging-example-app ./main.go

FROM scratch
COPY --from=build /bin/go-logging-example-app /go-logging-example-app
EXPOSE 8080
CMD ["/go-logging-example-app"]
