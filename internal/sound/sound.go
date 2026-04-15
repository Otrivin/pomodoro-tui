// Package sound plays a short notification sound using a pure-Go audio
// pipeline: the FLAC bytes embedded in the binary are decoded with
// github.com/gopxl/beep/v2/flac and streamed through oto, which binds to
// ALSA on Linux, CoreAudio on macOS and WASAPI on Windows. The resulting
// binary has no runtime prerequisites beyond the OS's native audio stack.
package sound

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/speaker"
)

var (
	initOnce sync.Once
	buffer   *beep.Buffer
)

// Init decodes the embedded FLAC bytes and prepares the speaker. The suffix
// argument is accepted for API compatibility with earlier versions but is
// unused — the decoder is hard-coded to FLAC.
func Init(audio []byte, _ string) {
	initOnce.Do(func() {
		if len(audio) == 0 {
			return
		}
		rc := io.NopCloser(bytes.NewReader(audio))
		streamer, format, err := flac.Decode(rc)
		if err != nil {
			return
		}
		defer streamer.Close()

		if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
			return
		}
		buf := beep.NewBuffer(format)
		buf.Append(streamer)
		buffer = buf
	})
}

// Ping plays the notification sound asynchronously. The terminal bell is
// always written as a last-resort fallback in case audio init failed.
func Ping() {
	fmt.Fprint(os.Stderr, "\a")
	if buffer == nil {
		return
	}
	speaker.Play(buffer.Streamer(0, buffer.Len()))
}

// Cleanup releases the audio device. Call on shutdown.
func Cleanup() {
	if buffer != nil {
		speaker.Close()
	}
}
