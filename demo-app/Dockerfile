FROM docker.io/library/golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN /usr/local/go/bin/go mod download

COPY . .
RUN /usr/local/go/bin/go version
RUN CGO_ENABLED=0 GOOS=linux /usr/local/go/bin/go build -trimpath -ldflags="-s -w" -o /demo-app ./main.go

FROM alpine:3.20
COPY --from=build /demo-app /usr/local/bin/demo-app
EXPOSE 8080
CMD ["/usr/local/bin/demo-app"]

