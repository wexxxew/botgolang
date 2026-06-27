package bot

import (
	"fmt"
	"math/rand/v2"

	"github.com/bwmarrin/discordgo"
)

const colorBlurple = 0x5865F2

// Ответы магического шара.
var eightBallAnswers = []string{
	"Бесспорно ✅", "Мне кажется — да", "Определённо да", "Можешь не сомневаться",
	"Скорее всего", "Хорошие перспективы", "Знаки говорят — да",
	"Пока неясно, попробуй ещё", "Спроси позже", "Лучше не рассказывать сейчас",
	"Даже не думай ❌", "Мой ответ — нет", "По моим данным — нет", "Очень сомнительно",
}

func (b *Bot) handleRoll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	sides := int64(6)
	if o, ok := optionMap(i)["sides"]; ok {
		sides = o.IntValue()
	}
	n := rand.Int64N(sides) + 1
	respond(s, i, fmt.Sprintf("🎲 Выпало: **%d** (из %d)", n, sides))
}

func (b *Bot) handle8ball(s *discordgo.Session, i *discordgo.InteractionCreate) {
	q := optionMap(i)["question"].StringValue()
	ans := eightBallAnswers[rand.IntN(len(eightBallAnswers))]
	respond(s, i, fmt.Sprintf("🎱 **Вопрос:** %s\n**Ответ:** %s", q, ans))
}

func (b *Bot) handleCoinflip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	side := "Орёл 🦅"
	if rand.IntN(2) == 0 {
		side = "Решка 🪙"
	}
	respond(s, i, "Подбрасываю монетку... **"+side+"**")
}

func (b *Bot) handleAvatar(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := invokerUser(i)
	if o, ok := optionMap(i)["user"]; ok {
		user = o.UserValue(s)
	}
	embed := &discordgo.MessageEmbed{
		Title: "Аватар " + user.Username,
		Image: &discordgo.MessageEmbedImage{URL: user.AvatarURL("512")},
		Color: colorBlurple,
	}
	respondEmbed(s, i, embed)
}

func (b *Bot) handleUserinfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := invokerUser(i)
	opt := optionMap(i)

	// Определяем участника сервера (для даты входа).
	var member *discordgo.Member
	if _, ok := opt["user"]; !ok {
		member = i.Member
	} else {
		user = opt["user"].UserValue(s)
		if data := i.ApplicationCommandData(); data.Resolved != nil {
			member = data.Resolved.Members[user.ID]
		}
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "ID", Value: user.ID, Inline: true},
		{Name: "Бот", Value: yesNo(user.Bot), Inline: true},
	}
	if created, err := discordgo.SnowflakeTimestamp(user.ID); err == nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Аккаунт создан",
			Value: fmt.Sprintf("<t:%d:D> (<t:%d:R>)", created.Unix(), created.Unix()),
		})
	}
	if member != nil && !member.JoinedAt.IsZero() {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Зашёл на сервер",
			Value: fmt.Sprintf("<t:%d:D> (<t:%d:R>)", member.JoinedAt.Unix(), member.JoinedAt.Unix()),
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:     user.Username,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: user.AvatarURL("256")},
		Color:     colorBlurple,
		Fields:    fields,
	}
	respondEmbed(s, i, embed)
}

func (b *Bot) handleServerinfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g, err := s.State.Guild(i.GuildID)
	if err != nil {
		g, err = s.Guild(i.GuildID)
		if err != nil {
			respondEphemeral(s, i, "❌ Не удалось получить информацию о сервере.")
			return
		}
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Участников", Value: fmt.Sprintf("%d", g.MemberCount), Inline: true},
		{Name: "Каналов", Value: fmt.Sprintf("%d", len(g.Channels)), Inline: true},
		{Name: "Владелец", Value: "<@" + g.OwnerID + ">", Inline: true},
		{Name: "ID", Value: g.ID, Inline: true},
	}
	if created, err := discordgo.SnowflakeTimestamp(g.ID); err == nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Создан",
			Value: fmt.Sprintf("<t:%d:D> (<t:%d:R>)", created.Unix(), created.Unix()),
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:     g.Name,
		Thumbnail: &discordgo.MessageEmbedThumbnail{URL: g.IconURL("256")},
		Color:     colorBlurple,
		Fields:    fields,
	}
	respondEmbed(s, i, embed)
}

func (b *Bot) handlePoll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	q := optionMap(i)["question"].StringValue()
	respond(s, i, "📊 **Опрос:** "+q+"\nГолосуй реакциями ниже 👇")

	// Добавляем реакции к опубликованному ответу.
	if msg, err := s.InteractionResponse(i.Interaction); err == nil {
		_ = s.MessageReactionAdd(msg.ChannelID, msg.ID, "✅")
		_ = s.MessageReactionAdd(msg.ChannelID, msg.ID, "❌")
	}
}

func yesNo(b bool) string {
	if b {
		return "да"
	}
	return "нет"
}
