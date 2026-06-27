package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"botgolang/internal/player"
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
	case "queue":
		b.handleQueue(s, i)
	case "leave":
		b.handleLeave(s, i)
	}
}

func (b *Bot) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	query := i.ApplicationCommandData().Options[0].StringValue()

	// Пользователь должен быть в голосовом канале.
	channelID := findUserVoiceChannel(s, i.GuildID, i.Member.User.ID)
	if channelID == "" {
		respond(s, i, "❌ Сначала зайди в голосовой канал.")
		return
	}

	// Подключение к голосу — долгая операция, отвечаем «думаю...».
	deferResponse(s, i)

	vc, err := s.ChannelVoiceJoin(i.GuildID, channelID, false, true)
	if err != nil {
		followup(s, i, "❌ Не удалось подключиться к голосовому каналу: "+err.Error())
		return
	}

	// Если это не ссылка — ищем на YouTube по названию.
	input := query
	if !strings.HasPrefix(query, "http://") && !strings.HasPrefix(query, "https://") {
		input = "ytsearch1:" + query
	}

	p := b.manager.Get(s, i.GuildID)
	pos := p.Enqueue(vc, i.ChannelID, player.Track{
		Input:       input,
		Title:       query,
		RequestedBy: i.Member.User.Username,
	})

	if pos == 1 {
		followup(s, i, "▶️ Запускаю: **"+query+"**")
	} else {
		followup(s, i, fmt.Sprintf("➕ Добавлено в очередь (#%d): **%s**", pos, query))
	}
}

func (b *Bot) handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.manager.Get(s, i.GuildID).SkipCurrent() {
		respond(s, i, "⏭️ Пропускаю текущий трек.")
	} else {
		respond(s, i, "❌ Сейчас ничего не играет.")
	}
}

func (b *Bot) handleStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.manager.Get(s, i.GuildID).StopAll() {
		respond(s, i, "⏹️ Останавливаю и очищаю очередь.")
	} else {
		respond(s, i, "❌ Сейчас ничего не играет.")
	}
}

func (b *Bot) handleQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	q := b.manager.Get(s, i.GuildID).Snapshot()
	if len(q) == 0 {
		respond(s, i, "📭 Очередь пуста.")
		return
	}
	var sb strings.Builder
	sb.WriteString("📜 **Очередь:**\n")
	for idx, t := range q {
		sb.WriteString(fmt.Sprintf("%d. %s\n", idx+1, t.Title))
	}
	respond(s, i, sb.String())
}

func (b *Bot) handleLeave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.manager.Get(s, i.GuildID).StopAll()
	respond(s, i, "👋 Выхожу из голосового канала.")
}

// findUserVoiceChannel ищет голосовой канал, в котором сейчас находится пользователь.
func findUserVoiceChannel(s *discordgo.Session, guildID, userID string) string {
	g, err := s.State.Guild(guildID)
	if err != nil {
		return ""
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID
		}
	}
	return ""
}

// --- помощники для ответов на интеракции ---

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
	if err != nil {
		log.Printf("ошибка ответа на интеракцию: %v", err)
	}
}

func deferResponse(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("ошибка deferred-ответа: %v", err)
	}
}

func followup(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &msg}); err != nil {
		log.Printf("ошибка followup-ответа: %v", err)
	}
}
