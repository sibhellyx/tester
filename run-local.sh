#!/bin/bash
set -e

mkdir -p ./logs

echo "▶ Поднимаем инфраструктуру..."
docker compose up -d postgres loki prometheus grafana promtail

echo "⏳ Ждём postgres..."
until docker exec postgres pg_isready -U postgres -d tester_db &>/dev/null; do
  sleep 1
done
echo "✔ Готово"

echo "▶ Запускаем сервис локально..."
export TZ=UTC
go run ./cmd/app/main.go 2>&1 | tee ./logs/tester.log