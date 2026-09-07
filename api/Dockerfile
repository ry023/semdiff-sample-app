FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/todo-api ./cmd/server

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app \
    && mkdir -p /data \
    && chown -R app:app /data
WORKDIR /app
COPY --from=build /out/todo-api /app/todo-api
USER app
EXPOSE 8080
ENTRYPOINT ["/app/todo-api"]

