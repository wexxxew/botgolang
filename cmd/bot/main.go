// Команда bot — точка входа музыкального Discord-бота.
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"botgolang/internal/bot"
)

func main() {
	// Подхватываем переменные из .env, если файл есть (для локального запуска).
	// В git файл .env не попадает — он в .gitignore.
	_ = godotenv.Load()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("не задан DISCORD_TOKEN (переменная окружения с токеном бота)")
	}

	b, err := bot.New(token)
	if err != nil {
		log.Fatalf("не удалось создать бота: %v", err)
	}

	if err := b.Run(); err != nil {
		log.Fatalf("ошибка работы бота: %v", err)
	}
}
