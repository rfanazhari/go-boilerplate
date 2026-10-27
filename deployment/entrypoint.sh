#!/bin/sh
set -e

if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

echo "running database migrations"
/app/migrate

echo "starting taskdemo"
exec /app/app
