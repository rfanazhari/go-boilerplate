# taskdemo

Go HTTP service for the Task Tracking reference feature.

## Enabled components

- PostgreSQL persistence (`pgx`, golang-migrate under `migrations/pgsql/`)
- RabbitMQ completion events (service-local AMQP adapter; eventmanager v0.2.5 publish errors are ignored by that library)
- logmanager JSON logging with credential masking
- go-i18n EN/ID error messages
- Unauthenticated `GET /health`

HTTP handlers are service-owned because httpmanager v0.16.11 health, 404, 405, and JSON decode paths do not use the required envelope or `DisallowUnknownFields`.

## Local setup

Full stack (Postgres, RabbitMQ, app + migrations):

```sh
docker compose up --build
```

The app listens on `http://127.0.0.1:8080`. Image build is `deployment/Dockerfile`.

Infrastructure only, then run the binary on the host:

```sh
docker compose up -d db rabbitmq
export DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/taskdemo?sslmode=disable
export RABBITMQ_URL=amqp://guest:guest@127.0.0.1:5672/
```

Copy `cmd/app/.env.example` for variable names. Do not commit secrets.

## Verification

```sh
go test ./...
make test
make test.arch
make test.smoke
make lint
make test.security
```

Optional live SQL/broker tests:

```sh
go test -tags=integration ./test/infrastructure/...
```

Those tests skip when PostgreSQL or RabbitMQ is unreachable.

## Task example

`POST /tasks` → `POST /tasks/{id}/start` → `POST /tasks/{id}/complete` → `GET /tasks/{id}`.

## Limitations

- No authentication or ownership
- No exactly-once event delivery
- Health does not check database or broker readiness
- Ordinary unit tests do not require Docker
