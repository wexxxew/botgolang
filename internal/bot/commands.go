package bot

import "github.com/bwmarrin/discordgo"

// Commands — список слэш-команд бота.
var Commands = []*discordgo.ApplicationCommand{
	// --- Музыка ---
	{
		Name:         "play",
		Description:  "Включить музыку (ссылка или название)",
		DMPermission: boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "query",
				Description: "Ссылка или название трека",
				Required:    true,
			},
		},
	},
	{Name: "skip", Description: "Пропустить текущий трек", DMPermission: boolPtr(false)},
	{Name: "stop", Description: "Остановить музыку и выйти из канала", DMPermission: boolPtr(false)},
	{Name: "pause", Description: "Пауза / продолжить", DMPermission: boolPtr(false)},
	{Name: "queue", Description: "Показать очередь треков", DMPermission: boolPtr(false)},

	// --- Развлечения и информация ---
	{Name: "ping", Description: "Проверка отклика бота"},
	{
		Name:        "roll",
		Description: "Бросить кубик",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "sides",
				Description: "Количество граней (по умолчанию 6)",
				Required:    false,
				MinValue:    floatPtr(2),
				MaxValue:    1000,
			},
		},
	},
	{
		Name:        "8ball",
		Description: "Магический шар предскажет ответ",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "question",
				Description: "Твой вопрос",
				Required:    true,
			},
		},
	},
	{Name: "coinflip", Description: "Подбросить монетку"},
	{
		Name:        "avatar",
		Description: "Показать аватар пользователя",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "Чей аватар (по умолчанию твой)",
				Required:    false,
			},
		},
	},
	{
		Name:        "userinfo",
		Description: "Информация о пользователе",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "О ком (по умолчанию ты)",
				Required:    false,
			},
		},
	},
	{Name: "serverinfo", Description: "Информация о сервере"},
	{
		Name:        "poll",
		Description: "Создать опрос (да/нет)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "question",
				Description: "Вопрос опроса",
				Required:    true,
			},
		},
	},
	{
		Name:        "vote",
		Description: "Голосование с вариантами — люди выбирают реакцией",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "question", Description: "Вопрос голосования", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "option1", Description: "Вариант 1", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "option2", Description: "Вариант 2", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "option3", Description: "Вариант 3 (необязательно)", Required: false},
			{Type: discordgo.ApplicationCommandOptionString, Name: "option4", Description: "Вариант 4 (необязательно)", Required: false},
		},
	},

	// --- Модерация (видны только тем, у кого есть права) ---
	{
		Name:                     "clear",
		Description:              "Удалить последние сообщения в канале",
		DefaultMemberPermissions: permPtr(discordgo.PermissionManageMessages),
		DMPermission:             boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "count",
				Description: "Сколько сообщений удалить (1–100)",
				Required:    true,
				MinValue:    floatPtr(1),
				MaxValue:    100,
			},
		},
	},
	{
		Name:                     "kick",
		Description:              "Выгнать участника",
		DefaultMemberPermissions: permPtr(discordgo.PermissionKickMembers),
		DMPermission:             boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Кого выгнать", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Причина", Required: false},
		},
	},
	{
		Name:                     "ban",
		Description:              "Забанить участника",
		DefaultMemberPermissions: permPtr(discordgo.PermissionBanMembers),
		DMPermission:             boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Кого забанить", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Причина", Required: false},
		},
	},
	{
		Name:                     "timeout",
		Description:              "Выдать тайм-аут участнику",
		DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
		DMPermission:             boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Кому", Required: true},
			{
				Type: discordgo.ApplicationCommandOptionInteger, Name: "minutes",
				Description: "На сколько минут (1–40320)", Required: true,
				MinValue: floatPtr(1), MaxValue: 40320,
			},
			{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Причина", Required: false},
		},
	},
}

// Вспомогательные функции для указателей (нужны полям ApplicationCommand).
func floatPtr(f float64) *float64 { return &f }
func boolPtr(b bool) *bool        { return &b }
func permPtr(p int64) *int64      { return &p }
