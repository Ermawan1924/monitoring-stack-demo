FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /demo-app main.go

FROM alpine:3.20
WORKDIR /app
COPY --from=build /demo-app /usr/local/bin/demo-app
EXPOSE 8080
CMD ["/usr/local/bin/demo-app"]