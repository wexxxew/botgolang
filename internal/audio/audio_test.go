package audio

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDemuxOggOpus проверяет, что демультиплексор извлекает Opus-кадры
// из реального Ogg-потока, сгенерированного ffmpeg.
// Тест пропускается, если ffmpeg недоступен.
func TestDemuxOggOpus(t *testing.T) {
	if _, err := exec.LookPath(ffmpegPath()); err != nil && !filepath.IsAbs(ffmpegPath()) {
		t.Skip("ffmpeg недоступен — пропускаю")
	}

	dir := t.TempDir()
	oggPath := filepath.Join(dir, "test.ogg")

	// 2 секунды синусоиды -> Opus 48 кГц стерео в Ogg.
	cmd := exec.Command(ffmpegPath(),
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-ar", "48000", "-ac", "2",
		"-c:a", "libopus", "-b:a", "96k",
		"-f", "ogg", oggPath,
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Skipf("не удалось запустить ffmpeg (%v) — пропускаю", err)
	}

	f, err := os.Open(oggPath)
	if err != nil {
		t.Fatalf("открыть %s: %v", oggPath, err)
	}
	defer f.Close()

	frames := make(chan []byte, 1024)
	done := make(chan error, 1)
	go func() {
		done <- demuxOggOpus(f, frames, make(chan struct{}), make(chan struct{}))
		close(frames)
	}()

	count := 0
	for range frames {
		count++
	}
	if err := <-done; err != nil {
		t.Fatalf("demuxOggOpus вернул ошибку: %v", err)
	}

	// 2 секунды по 20 мс = ~100 кадров. Допускаем погрешность.
	if count < 80 || count > 120 {
		t.Fatalf("ожидалось ~100 Opus-кадров, получено %d", count)
	}
	t.Logf("извлечено %d Opus-кадров — ок", count)
}
