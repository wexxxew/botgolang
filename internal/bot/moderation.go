package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleClear(s *discordgo.Session, i *discordgo.InteractionCreate) {
	count := optionMap(i)["count"].IntValue()

	msgs, err := s.ChannelMessages(i.ChannelID, int(count), "", "", "")
	if err != nil {
		respondEphemeral(s, i, "❌ Не удалось получить сообщения: "+err.Error())
		return
	}

	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}

	if err := s.ChannelMessagesBulkDelete(i.ChannelID, ids); err != nil {
		respondEphemeral(s, i, "❌ Не удалось удалить: "+err.Error()+
			"\n(Discord не даёт массово удалять сообщения старше 14 дней.)")
		return
	}

	respondEphemeral(s, i, fmt.Sprintf("🧹 Удалено сообщений: **%d**", len(ids)))
}

func (b *Bot) handleKick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opt := optionMap(i)
	user := opt["user"].UserValue(s)
	reason := optionalString(opt, "reason")

	if err := s.GuildMemberDeleteWithReason(i.GuildID, user.ID, reason); err != nil {
		respondEphemeral(s, i, "❌ Не удалось выгнать: "+err.Error())
		return
	}
	respond(s, i, fmt.Sprintf("👢 Выгнан <@%s>%s", user.ID, reasonSuffix(reason)))
}

func (b *Bot) handleBan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opt := optionMap(i)
	user := opt["user"].UserValue(s)
	reason := optionalString(opt, "reason")

	// 0 — не удалять историю сообщений забаненного.
	if err := s.GuildBanCreateWithReason(i.GuildID, user.ID, reason, 0); err != nil {
		respondEphemeral(s, i, "❌ Не удалось забанить: "+err.Error())
		return
	}
	respond(s, i, fmt.Sprintf("🔨 Забанен <@%s>%s", user.ID, reasonSuffix(reason)))
}

func (b *Bot) handleTimeout(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opt := optionMap(i)
	user := opt["user"].UserValue(s)
	minutes := opt["minutes"].IntValue()
	reason := optionalString(opt, "reason")

	until := time.Now().Add(time.Duration(minutes) * time.Minute)
	if err := s.GuildMemberTimeout(i.GuildID, user.ID, &until); err != nil {
		respondEphemeral(s, i, "❌ Не удалось выдать тайм-аут: "+err.Error())
		return
	}
	respond(s, i, fmt.Sprintf("🤐 <@%s> получил тайм-аут на **%d мин**%s", user.ID, minutes, reasonSuffix(reason)))
}

// --- помощники ---

func optionalString(opt map[string]*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	if o, ok := opt[name]; ok {
		return o.StringValue()
	}
	return ""
}

func reasonSuffix(reason string) string {
	if reason == "" {
		return ""
	}
	return " — причина: " + reason
}
