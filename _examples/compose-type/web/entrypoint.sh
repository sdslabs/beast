#!/bin/bash

until pg_isready -h 0.0.0.0 -p 5432 -U postgres; do
  echo "Waiting for database..."
  sleep 1
done

gunicorn --chdir /app --bind 0.0.0.0:80 app.main:app --access-logfile - --error-logfile -
