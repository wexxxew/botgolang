// Package player хранит очередь и управляет воспроизведением — по одному
// независимому проигрывателю на каждый сервер (guild).
package player

import (
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"

	"botgolang/internal/audio"
)

// Track — одна позиция в очереди.
type Track struct {
	Input       string // ссылка или поисковый запрос для yt-dlp
	Title       string // как показывать пользователю
	RequestedBy string
}

// Player — состояние воспроизведения одного сервера.
type Player struct {
	session       *discordgo.Session
	guildID       string
	textChannelID string

	mu      sync.Mutex
	queue   []Track
	vc      *discordgo.VoiceConnection
	playing bool
	skip    chan struct{} // закрывается для пропуска текущего трека
	stop    chan struct{} // закрывается для полной остановки
}

// Manager раздаёт по одному Player на сервер.
type Manager struct {
	mu      sync.Mutex
	players map[string]*Player
}

func NewManager() *Manager {
	return &Manager{players: make(map[string]*Player)}
}

// Get возвращает (создавая при необходимости) проигрыватель сервера.
func (m *Manager) Get(s *discordgo.Session, guildID string) *Player {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.players[guildID]
	if !ok {
		p = &Player{session: s, guildID: guildID}
		m.players[guildID] = p
	}
	return p
}

// Enqueue добавляет трек и при необходимости запускает цикл воспроизведения.
// Возвращает позицию трека в очереди (1 — играет сейчас).
func (p *Player) Enqueue(vc *discordgo.VoiceConnection, textChannelID string, t Track) (position int) {
	p.mu.Lock()
	p.vc = vc
	p.textChannelID = textChannelID
	p.queue = append(p.queue, t)
	position = len(p.queue)
	start := !p.playing
	if start {
		p.playing = true
		p.stop = make(chan struct{})
	}
	p.mu.Unlock()

	if start {
		go p.run()
	}
	return position
}

// run — основной цикл: берёт треки из очереди и проигрывает по очереди.
func (p *Player) run() {
	for {
		p.mu.Lock()
		if len(p.queue) == 0 {
			p.playing = false
			p.mu.Unlock()
			break
		}
		t := p.queue[0]
		p.queue = p.queue[1:]
		p.skip = make(chan struct{})
		skip, stop, vc := p.skip, p.stop, p.vc
		p.mu.Unlock()

		p.announce("🎶 Сейчас играет: **" + t.Title + "** (заказал " + t.RequestedBy + ")")

		err := audio.StreamTrack(vc, t.Input, skip, stop)
		if err == audio.ErrStopped {
			break
		}
		if err != nil && err != audio.ErrSkipped {
			log.Printf("[%s] ошибка воспроизведения %q: %v", p.guildID, t.Input, err)
			p.announce("⚠️ Не удалось проиграть **" + t.Title + "**: " + err.Error())
		}

		select {
		case <-stop:
			return
		default:
		}
	}

	// Очередь пуста или остановлено — выходим из голосового канала.
	p.mu.Lock()
	vc := p.vc
	p.vc = nil
	p.mu.Unlock()
	if vc != nil {
		_ = vc.Disconnect()
	}
	p.announce("⏹️ Очередь закончилась, выхожу из канала.")
}

// SkipCurrent пропускает текущий трек.
func (p *Player) SkipCurrent() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.playing || p.skip == nil {
		return false
	}
	closeOnce(p.skip)
	return true
}

// StopAll останавливает воспроизведение и очищает очередь.
func (p *Player) StopAll() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.playing {
		return false
	}
	p.queue = nil
	if p.stop != nil {
		closeOnce(p.stop)
	}
	return true
}

// Snapshot возвращает копию текущей очереди (для команды /queue).
func (p *Player) Snapshot() []Track {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Track, len(p.queue))
	copy(out, p.queue)
	return out
}

func (p *Player) announce(msg string) {
	if p.textChannelID == "" {
		return
	}
	if _, err := p.session.ChannelMessageSend(p.textChannelID, msg); err != nil {
		log.Printf("не удалось отправить сообщение: %v", err)
	}
}

// closeOnce безопасно закрывает канал, даже если он уже закрыт.
func closeOnce(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}
