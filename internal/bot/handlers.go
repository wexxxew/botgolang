package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// onInteraction маршрутизирует слэш-команды.
func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	switch i.ApplicationCommandData().Name {
	case "ping":
		respond(s, i, "🏓 pong!")
	case "play":
		b.handlePlay(s, i)
	case "skip":
		b.handleSkip(s, i)
	case "stop":
		b.handleStop(s, i)
	case "pause":
		b.handlePause(s, i)
	case "queue":
		b.handleQueue(s, i)
	case "roll":
		b.handleRoll(s, i)
	case "8ball":
		b.handle8ball(s, i)
	case "coinflip":
		b.handleCoinflip(s, i)
	case "avatar":
		b.handleAvatar(s, i)
	case "userinfo":
		b.handleUserinfo(s, i)
	case "serverinfo":
		b.handleServerinfo(s, i)
	case "poll":
		b.handlePoll(s, i)
	case "vote":
		b.handleVote(s, i)
	case "clear":
		b.handleClear(s, i)
	case "kick":
		b.handleKick(s, i)
	case "ban":
		b.handleBan(s, i)
	case "timeout":
		b.handleTimeout(s, i)
	}
}

// --- помощники ---

// optionMap превращает список опций команды в удобную карту по имени.
func optionMap(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	opts := i.ApplicationCommandData().Options
	m := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(opts))
	for _, o := range opts {
		m[o.Name] = o
	}
	return m
}

// invokerUser возвращает пользователя, вызвавшего команду (работает и на сервере, и в ЛС).
func invokerUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
	if err != nil {
		log.Printf("ошибка ответа на интеракцию: %v", err)
	}
}

// respondEphemeral отвечает сообщением, видимым только вызвавшему (для модерации).
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		log.Printf("ошибка ответа на интеракцию: %v", err)
	}
}

func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	if err != nil {
		log.Printf("ошибка ответа на интеракцию: %v", err)
	}
}

// deferResponse показывает «бот думает...» для долгих операций (например, /play).
func deferResponse(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("ошибка deferred-ответа: %v", err)
	}
}

// editResponse редактирует ранее отложенный (deferred) ответ.
func editResponse(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg}); err != nil {
		log.Printf("ошибка редактирования ответа: %v", err)
	}
}
