FROM golang:1.26-alpine AS build

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/app \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 65532 appuser

WORKDIR /app

COPY --from=build /out/app /app/app
COPY --from=build /out/migrate /app/migrate
COPY migrations /app/migrations
COPY deployment/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh \
 && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
