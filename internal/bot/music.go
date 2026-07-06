package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/snowflake/v2"

	"github.com/disgoorg/disgolink/v4/disgolink"
	"github.com/disgoorg/disgolink/v4/lavalink"
)

var (
	urlPattern    = regexp.MustCompile(`^https?://`)
	searchPattern = regexp.MustCompile(`^(.{2})search:(.+)`)
)

func lavalinkAddress() string {
	if a := os.Getenv("LAVALINK_ADDRESS"); a != "" {
		return a
	}
	// Именно 127.0.0.1, а не localhost: иначе Windows может увести в IPv6 (::1),
	// где Lavalink (слушает IPv4 0.0.0.0) недоступен.
	return "127.0.0.1:2333"
}

func lavalinkPassword() string {
	if p := os.Getenv("LAVALINK_PASSWORD"); p != "" {
		return p
	}
	return "youshallnotpass"
}

// setupLavalink подключается к серверу Lavalink. При успехе сохраняет клиент
// в b.lavalink; при ошибке оставляет nil (музыка будет недоступна).
func (b *Bot) setupLavalink() error {
	client := disgolink.New(snowflake.MustParse(b.session.State.User.ID),
		disgolink.WithListenerFunc(b.onTrackEnd),
		disgolink.WithListenerFunc(b.onTrackException),
		disgolink.WithListenerFunc(b.onTrackStuck),
		disgolink.WithListenerFunc(b.onWebSocketClosed),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.AddNode(ctx, disgolink.NodeConfig{
		Name:     "main",
		Address:  lavalinkAddress(),
		Password: lavalinkPassword(),
		Secure:   false,
	}); err != nil {
		return err
	}

	b.lavalink = client
	return nil
}

// --- проброс голосовых событий discordgo -> Lavalink ---

func (b *Bot) onVoiceStateUpdate(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	if b.lavalink == nil || e.UserID != s.State.User.ID {
		return
	}
	var channelID *snowflake.ID
	if e.ChannelID != "" {
		id := snowflake.MustParse(e.ChannelID)
		channelID = &id
	}
	b.lavalink.OnVoiceStateUpdate(context.TODO(), snowflake.MustParse(e.GuildID), channelID, e.SessionID)
	if e.ChannelID == "" {
		b.queues.delete(e.GuildID)
	}
}

func (b *Bot) onVoiceServerUpdate(s *discordgo.Session, e *discordgo.VoiceServerUpdate) {
	if b.lavalink == nil {
		return
	}
	b.lavalink.OnVoiceServerUpdate(context.TODO(), snowflake.MustParse(e.GuildID), e.Token, e.Endpoint)
}

// Логируем проблемы воспроизведения — помогает понять, почему «тишина».
func (b *Bot) onTrackException(e *disgolink.PlayerTrackExceptionEvent) {
	log.Printf("⚠️  ошибка трека: %+v", e)
}

func (b *Bot) onTrackStuck(e *disgolink.PlayerTrackStuckEvent) {
	log.Printf("⚠️  трек завис (нет данных): %+v", e)
}

func (b *Bot) onWebSocketClosed(e *disgolink.PlayerWebSocketClosedEvent) {
	log.Printf("⚠️  голосовое соединение закрыто: %+v", e)
}

// onTrackEnd автоматически запускает следующий трек из очереди.
func (b *Bot) onTrackEnd(e *disgolink.PlayerTrackEndEvent) {
	if !e.Reason.MayStartNext() {
		return
	}
	next, ok := b.queues.get(e.GetGuildID().String()).next()
	if !ok {
		return
	}
	if err := e.Player.Update(context.TODO(), disgolink.WithTrack(next)); err != nil {
		log.Printf("не удалось проиграть следующий трек: %v", err)
	}
}

// --- команды ---

func (b *Bot) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.lavalink == nil {
		respondEphemeral(s, i, "🎵 Музыка недоступна: сервер Lavalink не запущен. Запусти Lavalink и перезапусти бота.")
		return
	}

	query := optionMap(i)["query"].StringValue()
	identifier := query
	if !urlPattern.MatchString(identifier) && !searchPattern.MatchString(identifier) {
		// Ищем на SoundCloud: YouTube блокирует запросы с серверных IP.
		identifier = "scsearch:" + query
	}

	vs, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
	if err != nil || vs.ChannelID == "" {
		respondEphemeral(s, i, "❌ Сначала зайди в голосовой канал.")
		return
	}

	deferResponse(s, i)

	player := b.lavalink.Player(snowflake.MustParse(i.GuildID))
	queue := b.queues.get(i.GuildID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var toPlay *lavalink.Track
	b.lavalink.BestNode().Rest.LoadTracksHandler(ctx, identifier, disgolink.NewTrackLoadingResultHandler(
		func(track lavalink.Track) {
			if player.Track == nil {
				toPlay = &track
				editResponse(s, i, "🎶 Играю: **"+track.Info.Title+"**")
			} else {
				queue.add(track)
				editResponse(s, i, "➕ В очередь: **"+track.Info.Title+"**")
			}
		},
		func(playlist lavalink.Playlist) {
			if player.Track == nil {
				toPlay = &playlist.Tracks[0]
				queue.add(playlist.Tracks[1:]...)
			} else {
				queue.add(playlist.Tracks...)
			}
			editResponse(s, i, fmt.Sprintf("➕ Плейлист **%s** (%d треков)", playlist.Info.Name, len(playlist.Tracks)))
		},
		func(tracks []lavalink.Track) {
			if player.Track == nil {
				toPlay = &tracks[0]
				editResponse(s, i, "🎶 Играю: **"+tracks[0].Info.Title+"**")
			} else {
				queue.add(tracks[0])
				editResponse(s, i, "➕ В очередь: **"+tracks[0].Info.Title+"**")
			}
		},
		func() {
			editResponse(s, i, "🔍 Ничего не найдено по запросу: "+query)
		},
		func(err error) {
			editResponse(s, i, "⚠️ Ошибка поиска: "+err.Error())
		},
	))

	if toPlay == nil {
		return
	}

	if err := s.ChannelVoiceJoinManual(i.GuildID, vs.ChannelID, false, true); err != nil {
		editResponse(s, i, "❌ Не удалось зайти в канал: "+err.Error())
		return
	}
	if err := player.Update(context.TODO(), disgolink.WithTrack(*toPlay)); err != nil {
		editResponse(s, i, "❌ Не удалось запустить трек: "+err.Error())
	}
}

func (b *Bot) handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.lavalink == nil {
		respondEphemeral(s, i, "🎵 Музыка недоступна (Lavalink не запущен).")
		return
	}
	player := b.lavalink.ExistingPlayer(snowflake.MustParse(i.GuildID))
	if player == nil {
		respond(s, i, "❌ Сейчас ничего не играет.")
		return
	}

	next, ok := b.queues.get(i.GuildID).next()
	if !ok {
		_ = s.ChannelVoiceJoinManual(i.GuildID, "", false, false)
		respond(s, i, "⏭️ Очередь пуста — остановлено.")
		return
	}
	if err := player.Update(context.TODO(), disgolink.WithTrack(next)); err != nil {
		respond(s, i, "❌ Ошибка: "+err.Error())
		return
	}
	respond(s, i, "⏭️ Пропущено. Играет: **"+next.Info.Title+"**")
}

func (b *Bot) handleStop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.lavalink == nil {
		respondEphemeral(s, i, "🎵 Музыка недоступна (Lavalink не запущен).")
		return
	}
	if b.lavalink.ExistingPlayer(snowflake.MustParse(i.GuildID)) == nil {
		respond(s, i, "❌ Сейчас ничего не играет.")
		return
	}
	b.queues.delete(i.GuildID)
	if err := s.ChannelVoiceJoinManual(i.GuildID, "", false, false); err != nil {
		respond(s, i, "❌ Ошибка: "+err.Error())
		return
	}
	respond(s, i, "⏹️ Остановлено, вышел из канала.")
}

func (b *Bot) handlePause(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.lavalink == nil {
		respondEphemeral(s, i, "🎵 Музыка недоступна (Lavalink не запущен).")
		return
	}
	player := b.lavalink.ExistingPlayer(snowflake.MustParse(i.GuildID))
	if player == nil {
		respond(s, i, "❌ Сейчас ничего не играет.")
		return
	}
	if err := player.Update(context.TODO(), disgolink.WithPaused(!player.Paused)); err != nil {
		respond(s, i, "❌ Ошибка: "+err.Error())
		return
	}
	if player.Paused {
		respond(s, i, "⏸️ Пауза.")
	} else {
		respond(s, i, "▶️ Продолжаю.")
	}
}

func (b *Bot) handleQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.lavalink == nil {
		respondEphemeral(s, i, "🎵 Музыка недоступна (Lavalink не запущен).")
		return
	}
	tracks := b.queues.get(i.GuildID).snapshot()
	if len(tracks) == 0 {
		respond(s, i, "📭 Очередь пуста.")
		return
	}
	msg := "📜 **Очередь:**\n"
	for idx, t := range tracks {
		msg += fmt.Sprintf("%d. %s\n", idx+1, t.Info.Title)
	}
	respond(s, i, msg)
}

// --- очередь (по одной на сервер) ---

type musicQueue struct {
	mu     sync.Mutex
	tracks []lavalink.Track
}

type queueManager struct {
	mu     sync.Mutex
	queues map[string]*musicQueue
}

func newQueueManager() *queueManager {
	return &queueManager{queues: make(map[string]*musicQueue)}
}

func (m *queueManager) get(guildID string) *musicQueue {
	m.mu.Lock()
	defer m.mu.Unlock()
	q, ok := m.queues[guildID]
	if !ok {
		q = &musicQueue{}
		m.queues[guildID] = q
	}
	return q
}

func (m *queueManager) delete(guildID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.queues, guildID)
}

func (q *musicQueue) add(tracks ...lavalink.Track) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tracks = append(q.tracks, tracks...)
}

func (q *musicQueue) next() (lavalink.Track, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.tracks) == 0 {
		return lavalink.Track{}, false
	}
	t := q.tracks[0]
	q.tracks = q.tracks[1:]
	return t, true
}

func (q *musicQueue) snapshot() []lavalink.Track {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]lavalink.Track, len(q.tracks))
	copy(out, q.tracks)
	return out
}
