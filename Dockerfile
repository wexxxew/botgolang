# syntax=docker/dockerfile:1

# --- Этап сборки: компилируем Go-бинарник ---
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bot ./cmd/bot

# --- Этап запуска: Java (для Lavalink) + бинарник бота ---
FROM eclipse-temurin:17-jre
WORKDIR /app

# Бот
COPY --from=build /out/bot /app/bot

# Конфиг Lavalink (сам jar в git не хранится — качаем при сборке образа)
COPY lavalink/application.yml /app/lavalink/application.yml
ADD https://github.com/lavalink-devs/Lavalink/releases/download/4.2.2/Lavalink.jar /app/lavalink/Lavalink.jar

# Скрипт, который поднимает Lavalink и затем бота
COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh /app/bot

# Токен передаётся через переменную окружения DISCORD_TOKEN в настройках хостинга
CMD ["/app/start.sh"]
