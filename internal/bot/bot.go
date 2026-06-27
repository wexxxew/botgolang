// Package bot собирает Discord-сессию, регистрирует команды и обработчики.
package bot

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"github.com/disgoorg/disgolink/v4/disgolink"
)

// Bot — обёртка над сессией discordgo и клиентом Lavalink.
type Bot struct {
	session  *discordgo.Session
	lavalink *disgolink.Client // nil, если Lavalink недоступен (музыка отключается)
	queues   *queueManager
}

// New создаёт бота и навешивает обработчики (но ещё не подключается).
func New(token string) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	// TrackVoice нужен, чтобы знать, в каком голосовом канале пользователь.
	s.State.TrackVoice = true

	// Интенты: серверы, голосовые состояния (для музыки), сообщения и их
	// содержимое (для команды !ping).
	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMessages |
		discordgo.IntentMessageContent

	b := &Bot{
		session: s,
		queues:  newQueueManager(),
	}

	s.AddHandler(b.onReady)
	s.AddHandler(b.onGuildCreate)
	s.AddHandler(b.onMessage)
	s.AddHandler(b.onInteraction)
	// Проброс голосовых событий в Lavalink.
	s.AddHandler(b.onVoiceStateUpdate)
	s.AddHandler(b.onVoiceServerUpdate)

	return b, nil
}

// Run подключается к Discord и работает до сигнала завершения (Ctrl+C).
func (b *Bot) Run() error {
	if err := b.session.Open(); err != nil {
		return err
	}
	defer b.session.Close()

	// Подключаемся к Lavalink. Если сервер не запущен — музыка просто будет
	// недоступна, а остальные команды продолжат работать.
	if err := b.setupLavalink(); err != nil {
		log.Printf("⚠️  Lavalink не подключён (%v) — музыкальные команды недоступны, остальное работает", err)
	} else {
		log.Println("✅ Lavalink подключён — музыка доступна")
	}

	log.Println("бот работает. Нажми Ctrl+C для выхода.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("завершение работы...")
	return nil
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("бот запущен как %s", r.User.Username)
}

// onGuildCreate регистрирует команды для каждого сервера — так они появляются мгновенно.
func (b *Bot) onGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, g.ID, Commands); err != nil {
		log.Printf("не удалось зарегистрировать команды на сервере %s: %v", g.ID, err)
		return
	}
	log.Printf("команды зарегистрированы на сервере %s", g.ID)
}

func (b *Bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content == "!ping" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "🏓 pong!")
	}
}
