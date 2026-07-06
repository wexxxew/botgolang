#!/usr/bin/env bash
set -e

echo "▶️  Запуск Lavalink..."
cd /app/lavalink
java -Xmx512m -jar Lavalink.jar &

echo "⏳ Ожидание готовности Lavalink (порт 2333)..."
for i in $(seq 1 90); do
	if (exec 3<>/dev/tcp/127.0.0.1/2333) 2>/dev/null; then
		exec 3>&- 3<&-
		echo "✅ Lavalink готов"
		break
	fi
	sleep 2
done

echo "▶️  Запуск бота..."
cd /app
exec ./bot
