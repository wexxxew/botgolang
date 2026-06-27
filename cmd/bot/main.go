// Команда bot — точка входа музыкального Discord-бота.
package main

import (
	"log"
	"os"

	"botgolang/internal/bot"
)

func main() {
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
