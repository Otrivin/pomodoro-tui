package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// maxConfigSize bounds how much we'll read from the config file. 64 KiB is far
// more than a well-formed config ever needs; anything larger is suspicious.
const maxConfigSize = 64 * 1024

type Config struct {
	Focus               time.Duration `json:"focus"`
	ShortBreak          time.Duration `json:"short_break"`
	LongBreak           time.Duration `json:"long_break"`
	PomodorosBeforeLong int           `json:"pomodoros_before_long"`
}

func Default() Config {
	return Config{
		Focus:               25 * time.Minute,
		ShortBreak:          5 * time.Minute,
		LongBreak:           15 * time.Minute,
		PomodorosBeforeLong: 4,
	}
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pomodoro-tui", "config.json"), nil
}

// Load reads the config file. If it doesn't exist, returns the default config
// and no error. Malformed or oversized files fall back to defaults too.
func Load() Config {
	p, err := Path()
	if err != nil {
		return Default()
	}
	f, err := os.Open(p)
	if err != nil {
		return Default()
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxConfigSize))
	if err != nil {
		return Default()
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Default()
	}
	c.sanitize()
	return c
}

// Save persists the config to disk, creating the parent directory if needed.
func Save(c Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (c *Config) sanitize() {
	d := Default()
	if c.Focus <= 0 {
		c.Focus = d.Focus
	}
	if c.ShortBreak <= 0 {
		c.ShortBreak = d.ShortBreak
	}
	if c.LongBreak <= 0 {
		c.LongBreak = d.LongBreak
	}
	if c.PomodorosBeforeLong <= 0 {
		c.PomodorosBeforeLong = d.PomodorosBeforeLong
	}
}

// ErrNoConfig indicates the config file was absent (non-fatal).
var ErrNoConfig = errors.New("config file not found")
