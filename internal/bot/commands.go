package bot

import "github.com/bwmarrin/discordgo"

// Commands — список слэш-команд бота.
var Commands = []*discordgo.ApplicationCommand{
	{Name: "ping", Description: "Проверка отклика бота"},
	{
		Name:        "play",
		Description: "Воспроизвести трек из YouTube (ссылка или название)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "query",
				Description: "Ссылка или поисковый запрос",
				Required:    true,
			},
		},
	},
	{Name: "skip", Description: "Пропустить текущий трек"},
	{Name: "stop", Description: "Остановить воспроизведение и очистить очередь"},
	{Name: "queue", Description: "Показать очередь"},
	{Name: "leave", Description: "Выйти из голосового канала"},
}
