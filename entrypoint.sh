#!/busybox/sh
set -e

# Запуск миграций
/bin/migrate -path /migrations -database "$DB_URL" up

# Запуск приложения
exec /app