#!/bin/sh

echo "Waiting for postgres..."

while ! nc -z $POSTGRES_HOST $POSTGRES_PORT; do
    sleep 3
done

echo "PostgreSQL started"
go run cmd/populate_db/main.go

exec "$@"
