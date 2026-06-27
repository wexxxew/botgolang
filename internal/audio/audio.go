// Package audio отвечает за конвейер звука:
// yt-dlp (загрузка) -> ffmpeg (перекодировка в Opus/Ogg) -> разбор Ogg -> Discord.
// Перекодировку делает ffmpeg, поэтому cgo не требуется.
package audio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bwmarrin/discordgo"
)

// Сигналы завершения воспроизведения трека.
var (
	ErrSkipped = errors.New("трек пропущен")
	ErrStopped = errors.New("воспроизведение остановлено")
)

// ytdlpPath / ffmpegPath берут бинарники из PATH, но позволяют переопределить
// путь через переменные окружения (удобно, пока PATH не обновился после установки).
func ytdlpPath() string {
	if p := os.Getenv("YTDLP_PATH"); p != "" {
		return p
	}
	return "yt-dlp"
}

func ffmpegPath() string {
	if p := os.Getenv("FFMPEG_PATH"); p != "" {
		return p
	}
	return "ffmpeg"
}

// StreamTrack проигрывает один трек в голосовом канале.
// Блокируется до конца трека, пропуска (skip) или остановки (stop).
func StreamTrack(vc *discordgo.VoiceConnection, input string, skip, stop <-chan struct{}) error {
	// yt-dlp вытаскивает лучшую аудиодорожку и пишет её в stdout.
	yt := exec.Command(ytdlpPath(),
		"-f", "bestaudio",
		"--no-playlist",
		"--quiet", "--no-warnings",
		"-o", "-",
		input,
	)

	// ffmpeg перекодирует поток в Opus (48 кГц, стерео) в Ogg-контейнере.
	ff := exec.Command(ffmpegPath(),
		"-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-vn",
		"-ar", "48000",
		"-ac", "2",
		"-c:a", "libopus",
		"-b:a", "96k",
		"-f", "ogg",
		"pipe:1",
	)

	ytOut, err := yt.StdoutPipe()
	if err != nil {
		return fmt.Errorf("yt-dlp stdout: %w", err)
	}
	ff.Stdin = ytOut

	ffOut, err := ff.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout: %w", err)
	}
	yt.Stderr = os.Stderr
	ff.Stderr = os.Stderr

	if err := yt.Start(); err != nil {
		return fmt.Errorf("не удалось запустить yt-dlp: %w", err)
	}
	if err := ff.Start(); err != nil {
		_ = yt.Process.Kill()
		return fmt.Errorf("не удалось запустить ffmpeg: %w", err)
	}

	defer func() {
		_ = yt.Process.Kill()
		_ = ff.Process.Kill()
		_ = yt.Wait()
		_ = ff.Wait()
	}()

	_ = vc.Speaking(true)
	defer vc.Speaking(false)

	// Темп (20 мс/кадр) держит сам discordgo: канал OpusSend
	// блокирует отправку до нужного момента — получаем естественный backpressure.
	return demuxOggOpus(ffOut, vc.OpusSend, skip, stop)
}

// demuxOggOpus читает Ogg-поток, выделяет Opus-пакеты и шлёт их в frames.
// Упрощение: считаем, что каждый аудиокадр умещается в одну Ogg-страницу
// (для Opus 20 мс при разумном битрейте это всегда так).
func demuxOggOpus(r io.Reader, frames chan<- []byte, skip, stop <-chan struct{}) error {
	header := make([]byte, 27)
	for {
		if _, err := io.ReadFull(r, header); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil // трек закончился
			}
			return err
		}
		if string(header[0:4]) != "OggS" {
			return errors.New("повреждённая Ogg-страница (нет сигнатуры OggS)")
		}

		segCount := int(header[26])
		segTable := make([]byte, segCount)
		if _, err := io.ReadFull(r, segTable); err != nil {
			return err
		}

		payloadLen := 0
		for _, s := range segTable {
			payloadLen += int(s)
		}
		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(r, payload); err != nil {
			return err
		}

		// Разбиваем payload на пакеты по таблице сегментов (Ogg lacing):
		// сегмент 255 продолжает пакет, сегмент <255 завершает его.
		offset, pktLen := 0, 0
		for _, s := range segTable {
			pktLen += int(s)
			if s == 255 {
				continue
			}
			pkt := payload[offset : offset+pktLen]
			offset += pktLen
			pktLen = 0

			// Пропускаем служебные заголовки Opus.
			if len(pkt) >= 8 {
				if h := string(pkt[0:8]); h == "OpusHead" || h == "OpusTags" {
					continue
				}
			}
			if len(pkt) == 0 {
				continue
			}

			// Копируем кадр: payload переиспользуется на следующей странице.
			frame := make([]byte, len(pkt))
			copy(frame, pkt)

			select {
			case frames <- frame:
			case <-skip:
				return ErrSkipped
			case <-stop:
				return ErrStopped
			}
		}
	}
}
