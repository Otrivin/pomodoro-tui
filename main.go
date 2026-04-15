package main

import (
	_ "embed"
	"flag"
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lg "charm.land/lipgloss/v2"

	"pomodoro-tui/internal/config"
	"pomodoro-tui/internal/digits"
	"pomodoro-tui/internal/sound"
)

//go:embed audio/653896__therealdeevee__submarine-sonar-ping.flac
var pingAudio []byte

// ─── domain ────────────────────────────────────────────────────────────────

type phase int

const (
	phaseWork phase = iota
	phaseShortBreak
	phaseLongBreak
)

func (p phase) label() string {
	switch p {
	case phaseWork:
		return "FOCUS"
	case phaseShortBreak:
		return "SHORT BREAK"
	default:
		return "LONG BREAK"
	}
}

func (m model) phaseDuration(p phase) time.Duration {
	switch p {
	case phaseWork:
		return m.cfg.Focus
	case phaseShortBreak:
		return m.cfg.ShortBreak
	default:
		return m.cfg.LongBreak
	}
}

// ─── colors / styles ───────────────────────────────────────────────────────

var (
	soupRed    = lg.Color("#C8102E")
	deepRed    = lg.Color("#8E0B20")
	gold       = lg.Color("#D4AF37")
	cream      = lg.Color("#F4EFE6")
	ink        = lg.Color("#1A1A1A")
	mint       = lg.Color("#7BD3A0")
	sky        = lg.Color("#7BB6E2")
	mutedGray  = lg.Color("#6B6B6B")
	softGray   = lg.Color("#A8A8A8")
	highlight  = lg.Color("#FFE17A")
	background = lg.Color("#000000")
)

// ─── model ─────────────────────────────────────────────────────────────────

type tickMsg time.Time

type model struct {
	phase       phase
	remaining   time.Duration
	running     bool
	completed   int
	width       int
	height      int
	lastTick    time.Time
	flashUntil  time.Time
	muted       bool
	showInfo    bool
	showOptions bool
	optionIdx   int
	cfg         config.Config
}

func initialModel() model {
	cfg := config.Load()
	m := model{
		phase: phaseWork,
		cfg:   cfg,
	}
	m.remaining = m.phaseDuration(phaseWork)
	return m
}

func (m model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// ─── update ────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyPressMsg:
		key := strings.ToLower(msg.String())
		if m.showOptions {
			return m.handleOptionsKey(key)
		}
		switch key {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case " ", "space", "enter":
			m.running = !m.running
			m.lastTick = time.Now()
			return m, nil
		case "r":
			m.remaining = m.phaseDuration(m.phase)
			m.running = false
			return m, nil
		case "s":
			m.advancePhase(false)
			return m, nil
		case "n":
			m.completed = 0
			m.phase = phaseWork
			m.remaining = m.phaseDuration(phaseWork)
			m.running = false
			return m, nil
		case "m":
			m.muted = !m.muted
			return m, nil
		case "i", "?":
			m.showInfo = !m.showInfo
			return m, nil
		case "o":
			m.showOptions = true
			m.showInfo = false
			m.optionIdx = 0
			return m, nil
		case "t":
			if !m.muted {
				sound.Ping()
			}
			m.flashUntil = time.Now().Add(800 * time.Millisecond)
			return m, nil
		}

	case tickMsg:
		now := time.Time(msg)
		if m.running {
			if m.lastTick.IsZero() {
				m.lastTick = now
			}
			delta := now.Sub(m.lastTick)
			m.lastTick = now
			m.remaining -= delta
			if m.remaining <= 0 {
				m.remaining = 0
				m.running = false
				if !m.muted {
					sound.Ping()
				}
				m.flashUntil = now.Add(1500 * time.Millisecond)
				m.advancePhase(true)
			}
		} else {
			m.lastTick = now
		}
		return m, tick()
	}
	return m, nil
}

const optionFieldCount = 4

func (m model) handleOptionsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c", "esc", "o":
		m.showOptions = false
		return m, nil
	case "up", "k":
		m.optionIdx--
		if m.optionIdx < 0 {
			m.optionIdx = optionFieldCount - 1
		}
		return m, nil
	case "down", "j", "tab":
		m.optionIdx = (m.optionIdx + 1) % optionFieldCount
		return m, nil
	case "left", "h", "-":
		m.adjustOption(-1)
		return m, nil
	case "right", "l", "+", "=":
		m.adjustOption(1)
		return m, nil
	case "d":
		m.cfg = config.Default()
		m.remaining = m.phaseDuration(m.phase)
		_ = config.Save(m.cfg)
		return m, nil
	}
	return m, nil
}

func (m *model) adjustOption(delta int) {
	step := time.Minute
	switch m.optionIdx {
	case 0:
		m.cfg.Focus = clampDuration(m.cfg.Focus+time.Duration(delta)*step, time.Minute, 180*time.Minute)
	case 1:
		m.cfg.ShortBreak = clampDuration(m.cfg.ShortBreak+time.Duration(delta)*step, time.Minute, 60*time.Minute)
	case 2:
		m.cfg.LongBreak = clampDuration(m.cfg.LongBreak+time.Duration(delta)*step, time.Minute, 120*time.Minute)
	case 3:
		v := m.cfg.PomodorosBeforeLong + delta
		if v < 2 {
			v = 2
		}
		if v > 12 {
			v = 12
		}
		m.cfg.PomodorosBeforeLong = v
	}
	// Keep the current phase's remaining time in sync with the new duration
	// when we're not running; never push remaining above the new total.
	if total := m.phaseDuration(m.phase); m.remaining > total || !m.running {
		m.remaining = total
	}
	_ = config.Save(m.cfg)
}

func clampDuration(d, lo, hi time.Duration) time.Duration {
	if d < lo {
		return lo
	}
	if d > hi {
		return hi
	}
	return d
}

func (m *model) advancePhase(fromTimer bool) {
	if fromTimer && m.phase == phaseWork {
		m.completed++
	}
	switch m.phase {
	case phaseWork:
		if m.completed%m.cfg.PomodorosBeforeLong == 0 && m.completed > 0 {
			m.phase = phaseLongBreak
		} else {
			m.phase = phaseShortBreak
		}
	default:
		m.phase = phaseWork
	}
	m.remaining = m.phaseDuration(m.phase)
	m.running = false
}

// ─── view ──────────────────────────────────────────────────────────────────

func (m model) View() tea.View {
	var body string
	switch {
	case m.showOptions:
		body = m.optionsPanel()
	case m.showInfo:
		body = m.infoPanel()
	default:
		right := lg.NewStyle().
			MarginLeft(4).
			MarginBackground(background).
			Background(background).
			Render(m.rightPanel())
		body = lg.JoinHorizontal(lg.Top, soupCan(), right)
	}

	body = lg.NewStyle().Background(background).Render(body)

	frame := lg.NewStyle().
		Padding(1, 3).
		Border(lg.RoundedBorder()).
		BorderForeground(deepRed).
		BorderBackground(background).
		Foreground(cream).
		Background(background).
		Render(body)

	w, h := m.width, m.height
	if w == 0 {
		w = lg.Width(frame) + 4
	}
	if h == 0 {
		h = lg.Height(frame) + 2
	}
	out := lg.Place(w, h, lg.Center, lg.Center, frame,
		lg.WithWhitespaceChars(" "),
		lg.WithWhitespaceStyle(lg.NewStyle().Background(background).Foreground(background)),
	)
	out = lg.NewStyle().Background(background).Render(out)
	// Every "reset all" escape reverts background to the terminal default.
	// Immediately re-assert our black background so no cell can leak.
	out = strings.ReplaceAll(out, "\x1b[0m", "\x1b[0m\x1b[48;2;0;0;0m")
	out = strings.ReplaceAll(out, "\x1b[m", "\x1b[m\x1b[48;2;0;0;0m")
	v := tea.NewView(out)
	v.AltScreen = true
	v.BackgroundColor = background
	return v
}

func (m model) optionsPanel() string {
	const width = 52
	pad := lg.NewStyle().Background(background).Width(width).Render
	label := lg.NewStyle().Foreground(softGray).Background(background).Width(22).Render
	accent := lg.NewStyle().Foreground(soupRed).Background(background).Bold(true).Render
	help := lg.NewStyle().Foreground(mutedGray).Background(background).Render

	title := lg.NewStyle().
		Bold(true).
		Foreground(cream).
		Background(soupRed).
		Padding(0, 2).
		MarginBottom(1).
		Render("⚙  OPTIONS")

	fields := []struct {
		name, value string
	}{
		{"focus", formatMins(m.cfg.Focus)},
		{"short break", formatMins(m.cfg.ShortBreak)},
		{"long break", formatMins(m.cfg.LongBreak)},
		{"long break after", fmt.Sprintf("%d pomodoros", m.cfg.PomodorosBeforeLong)},
	}

	row := func(i int, name, value string) string {
		selected := i == m.optionIdx
		arrowL := " "
		arrowR := " "
		marker := "  "
		valFg := cream
		if selected {
			arrowL = "◂"
			arrowR = "▸"
			marker = "▸ "
			valFg = highlight
		}
		markerStyle := lg.NewStyle().Foreground(soupRed).Background(background).Bold(true).Render(marker)
		arrowStyle := lg.NewStyle().Foreground(soupRed).Background(background).Bold(true)
		valStyle := lg.NewStyle().Foreground(valFg).Background(background).Bold(true)
		return markerStyle +
			label(name) +
			arrowStyle.Render(arrowL+" ") +
			valStyle.Render(value) +
			arrowStyle.Render(" "+arrowR)
	}

	path, _ := config.Path()
	if path == "" {
		path = "(none)"
	}

	rows := []string{
		pad(title),
		pad(accent("Durations and cycle")),
		pad(""),
	}
	for i, f := range fields {
		rows = append(rows, pad(row(i, f.name, f.value)))
	}
	rows = append(rows,
		pad(""),
		pad(help("↑/↓ select   ←/→ adjust   d defaults   o/esc close")),
		pad(""),
		pad(help("saved to: "+path)),
	)

	return lg.JoinVertical(lg.Left, rows...)
}

func formatMins(d time.Duration) string {
	m := int(d / time.Minute)
	if m == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", m)
}

func (m model) infoPanel() string {
	label := lg.NewStyle().Foreground(softGray).Background(background).Width(24).Render
	val := lg.NewStyle().Foreground(cream).Background(background).Bold(true).Render
	accent := lg.NewStyle().Foreground(soupRed).Background(background).Bold(true).Render
	pad := lg.NewStyle().Background(background).Width(44).Render

	current := m.phase.label()
	nextPhase := phaseShortBreak
	if m.phase == phaseWork {
		if (m.completed+1)%m.cfg.PomodorosBeforeLong == 0 {
			nextPhase = phaseLongBreak
		}
	} else {
		nextPhase = phaseWork
	}

	mute := "off"
	if m.muted {
		mute = "on"
	}

	title := lg.NewStyle().
		Bold(true).
		Foreground(cream).
		Background(soupRed).
		Padding(0, 2).
		MarginBottom(1).
		Render("ⓘ  CONFIGURATION")

	rows := []string{
		pad(title),
		pad(accent("Durations")),
		pad(label("  focus") + val(m.phaseDuration(phaseWork).String())),
		pad(label("  short break") + val(m.phaseDuration(phaseShortBreak).String())),
		pad(label("  long break") + val(m.phaseDuration(phaseLongBreak).String())),
		pad(label("  long break after") + val(fmt.Sprintf("%d pomodoros", m.cfg.PomodorosBeforeLong))),
		pad(""),
		pad(accent("Current cycle")),
		pad(label("  phase") + val(current)),
		pad(label("  remaining") + val(formatDuration(m.remaining))),
		pad(label("  pomodoros completed") + val(fmt.Sprintf("%d", m.completed))),
		pad(label("  position in cycle") + val(fmt.Sprintf("%d / %d", m.completed%m.cfg.PomodorosBeforeLong, m.cfg.PomodorosBeforeLong))),
		pad(label("  next phase") + val(nextPhase.label())),
		pad(""),
		pad(accent("Audio")),
		pad(label("  ping on timer end") + val(map[bool]string{true: "muted", false: "enabled"}[m.muted])),
		pad(label("  mute") + val(mute)),
		pad(""),
		pad(lg.NewStyle().Foreground(mutedGray).Background(background).Render("press i to close")),
	}

	return lg.JoinVertical(lg.Left, rows...)
}

func (m model) phaseAccent() color.Color {
	switch m.phase {
	case phaseShortBreak:
		return mint
	case phaseLongBreak:
		return sky
	default:
		return soupRed
	}
}

func (m model) rightPanel() string {
	accent := m.phaseAccent()
	const panelW = 46
	bg := lg.NewStyle().Background(background)

	banner := lg.NewStyle().
		Bold(true).
		Foreground(accent).
		Background(background).
		Width(panelW).
		Align(lg.Center).
		Render("P · O · M · O · D · O · R · O")
	bannerRule := lg.NewStyle().
		Foreground(accent).
		Background(background).
		Width(panelW).
		Align(lg.Center).
		Render(strings.Repeat("━", panelW-2))

	badge := lg.NewStyle().
		Bold(true).
		Foreground(cream).
		Background(accent).
		Padding(0, 2).
		Render("● " + m.phase.label())
	nextLabel := m.nextPhase().label()
	next := lg.NewStyle().
		Foreground(softGray).
		Background(background).
		Italic(true).
		Render("next · " + nextLabel)
	phaseRow := lg.NewStyle().
		Width(panelW).
		Background(background).
		Render(lg.JoinHorizontal(lg.Center, badge, bg.Render("   "), next))

	timeStr := formatDuration(m.remaining)
	timerColor := cream
	switch {
	case m.remaining == 0:
		timerColor = highlight
	case m.remaining < 60*time.Second && m.running:
		timerColor = highlight
	case !m.flashUntil.IsZero() && time.Now().Before(m.flashUntil):
		timerColor = highlight
	}
	timerBody := lg.NewStyle().
		Foreground(timerColor).
		Background(background).
		Bold(true).
		Render(digits.Render(timeStr))
	timerBox := lg.NewStyle().
		Border(lg.ThickBorder()).
		BorderForeground(accent).
		BorderBackground(background).
		Background(background).
		Padding(0, 3).
		Width(panelW - 2).
		Align(lg.Center).
		Render(timerBody)

	pipsBox := lg.NewStyle().
		Border(lg.NormalBorder()).
		BorderForeground(lg.Color("#2E2E35")).
		BorderBackground(background).
		Background(background).
		Padding(0, 2).
		Width(panelW - 2).
		Align(lg.Center).
		Render(m.pips())

	// bar consumes: 2 caps + body. Leave room for "  " + "XXX%" (6 cells).
	barBodyW := panelW - 2 - 2 - 2 - 4
	bar := m.progressBar(barBodyW)
	pct := 0
	if total := m.phaseDuration(m.phase); total > 0 {
		frac := 1 - float64(m.remaining)/float64(total)
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		pct = int(frac * 100)
	}
	pctStr := lg.NewStyle().Foreground(softGray).Background(background).Render(fmt.Sprintf("%3d%%", pct))
	barRow := lg.NewStyle().
		Width(panelW).
		Background(background).
		Render(bar + bg.Render("  ") + pctStr)

	statusText := "PAUSED"
	statusFg := softGray
	statusDot := "◐"
	if m.running {
		statusText = "RUNNING"
		statusFg = mint
		statusDot = "●"
	}
	if m.remaining == 0 {
		statusText = "DONE"
		statusFg = highlight
		statusDot = "✦"
	}
	status := lg.NewStyle().Foreground(statusFg).Background(background).Bold(true).Render(statusDot + " " + statusText)

	muteIcon := "sound on"
	muteFg := mint
	muteGlyph := "♪"
	if m.muted {
		muteIcon = "muted"
		muteFg = softGray
		muteGlyph = "∅"
	}
	muteBadge := lg.NewStyle().Foreground(muteFg).Background(background).Render(muteGlyph + " " + muteIcon)

	cycleCount := lg.NewStyle().Foreground(softGray).Background(background).Render(
		fmt.Sprintf("✓ %d completed", m.completed),
	)
	statsRow := lg.NewStyle().
		Width(panelW).
		Background(background).
		Render(lg.JoinHorizontal(lg.Left, status, bg.Render("    "), cycleCount, bg.Render("    "), muteBadge))

	help := lg.NewStyle().
		Width(panelW).
		Background(background).
		Render(renderKeyHints())

	return lg.JoinVertical(lg.Left,
		banner,
		bannerRule,
		bg.Render(strings.Repeat(" ", panelW)),
		phaseRow,
		bg.Render(strings.Repeat(" ", panelW)),
		timerBox,
		bg.Render(strings.Repeat(" ", panelW)),
		barRow,
		bg.Render(strings.Repeat(" ", panelW)),
		pipsBox,
		bg.Render(strings.Repeat(" ", panelW)),
		statsRow,
		bg.Render(strings.Repeat(" ", panelW)),
		help,
	)
}

func (m model) nextPhase() phase {
	if m.phase != phaseWork {
		return phaseWork
	}
	if (m.completed+1)%m.cfg.PomodorosBeforeLong == 0 {
		return phaseLongBreak
	}
	return phaseShortBreak
}

func renderKeyHints() string {
	bg := lg.NewStyle().Background(background)
	gap := bg.Render("  ")
	chip := func(key, label string) string {
		k := lg.NewStyle().
			Background(lg.Color("#2A2A33")).
			Foreground(highlight).
			Bold(true).
			Render(" " + key + " ")
		l := lg.NewStyle().
			Foreground(softGray).
			Background(background).
			Render(" " + label + " ")
		return k + l
	}

	row1 := chip("space", "start/pause") + gap + chip("r", "reset")
	row2 := chip("s", "skip") + gap + chip("n", "new cycle")
	row3 := chip("i", "info") + gap + chip("o", "options") + gap + chip("t", "test") + gap + chip("m", "mute") + gap + chip("q", "quit")
	return lg.JoinVertical(lg.Left, row1, row2, row3)
}

func (m model) pips() string {
	filled := lg.NewStyle().Foreground(soupRed).Background(background).Render("●")
	empty := lg.NewStyle().Foreground(mutedGray).Background(background).Render("○")
	current := lg.NewStyle().Foreground(highlight).Background(background).Render("◉")
	sep := lg.NewStyle().Background(background).Render("  ")

	done := m.completed % m.cfg.PomodorosBeforeLong
	parts := make([]string, m.cfg.PomodorosBeforeLong)
	for i := 0; i < m.cfg.PomodorosBeforeLong; i++ {
		switch {
		case i < done:
			parts[i] = filled
		case i == done && m.phase == phaseWork:
			parts[i] = current
		default:
			parts[i] = empty
		}
	}
	label := lg.NewStyle().Foreground(softGray).Background(background).Render("cycle  ")
	return label + strings.Join(parts, sep)
}

func (m model) progressBar(width int) string {
	total := m.phaseDuration(m.phase)
	if total == 0 {
		return ""
	}
	frac := 1 - float64(m.remaining)/float64(total)
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}

	exact := float64(width) * frac
	filledW := int(exact)
	partial := exact - float64(filledW)
	partialGlyphs := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	partialIdx := int(partial * 8)
	if partialIdx > 7 {
		partialIdx = 7
	}

	accent := m.phaseAccent()
	trackCol := lg.Color("#2A2A33")

	leftCap := lg.NewStyle().Foreground(accent).Background(background).Render("▐")
	rightCap := lg.NewStyle().Foreground(trackCol).Background(background).Render("▌")

	full := lg.NewStyle().Foreground(accent).Background(background).Render(strings.Repeat("█", filledW))
	rest := width - filledW
	var leading string
	if rest > 0 && partialIdx > 0 {
		leading = lg.NewStyle().
			Foreground(accent).
			Background(trackCol).
			Render(partialGlyphs[partialIdx])
		rest--
	}
	track := lg.NewStyle().Foreground(trackCol).Background(background).Render(strings.Repeat("█", rest))

	return leftCap + full + leading + track + rightCap
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Round(time.Second).Seconds())
	mins := total / 60
	secs := total % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

// ─── soup can art ──────────────────────────────────────────────────────────

func soupCan() string {
	red := lg.NewStyle().Background(soupRed).Foreground(cream).Bold(true)
	redPlain := lg.NewStyle().Background(soupRed).Foreground(cream)
	goldBand := lg.NewStyle().Background(gold).Foreground(ink).Bold(true)
	creamBg := lg.NewStyle().Background(cream).Foreground(soupRed).Bold(true)
	creamPlain := lg.NewStyle().Background(cream).Foreground(soupRed)
	rim := lg.NewStyle().Foreground(deepRed).Background(background)

	const w = 22
	pad := func(s string, style lg.Style) string {
		visW := lg.Width(s)
		if visW > w {
			s = s[:w]
			visW = w
		}
		left := (w - visW) / 2
		right := w - visW - left
		return style.Render(strings.Repeat(" ", left) + s + strings.Repeat(" ", right))
	}

	rows := []string{
		rim.Render("  ╭" + strings.Repeat("─", w) + "╮  "),
		rim.Render("  │") + redPlain.Render(strings.Repeat(" ", w)) + rim.Render("│  "),
		rim.Render("  │") + pad("~ Campbell's ~", red) + rim.Render("│  "),
		rim.Render("  │") + redPlain.Render(strings.Repeat(" ", w)) + rim.Render("│  "),
		rim.Render("  │") + pad("★  ★  ★  ★", goldBand) + rim.Render("│  "),
		rim.Render("  │") + creamPlain.Render(strings.Repeat(" ", w)) + rim.Render("│  "),
		rim.Render("  │") + pad("CONDENSED", creamBg) + rim.Render("│  "),
		rim.Render("  │") + creamPlain.Render(strings.Repeat(" ", w)) + rim.Render("│  "),
		rim.Render("  │") + pad("TOMATO", creamBg) + rim.Render("│  "),
		rim.Render("  │") + pad("SOUP", creamBg) + rim.Render("│  "),
		rim.Render("  │") + creamPlain.Render(strings.Repeat(" ", w)) + rim.Render("│  "),
		rim.Render("  │") + pad("★  ★  ★  ★", goldBand) + rim.Render("│  "),
		rim.Render("  │") + redPlain.Render(strings.Repeat(" ", w)) + rim.Render("│  "),
		rim.Render("  ╰" + strings.Repeat("─", w) + "╯  "),
	}

	can := lg.JoinVertical(lg.Left, rows...)

	shadowW := w + 4
	pedestalStyle := lg.NewStyle().Foreground(lg.Color("#2A1014")).Background(background)
	hazeStyle := lg.NewStyle().Foreground(lg.Color("#15080B")).Background(background)
	pedestal := pedestalStyle.Render(" " + strings.Repeat("▀", shadowW) + " ")
	haze := hazeStyle.Render("  " + strings.Repeat("·", shadowW-2) + "  ")

	return lg.JoinVertical(lg.Center, can, pedestal, haze)
}

// ─── main ──────────────────────────────────────────────────────────────────

func main() {
	snapshot := flag.Bool("snapshot", false, "render current view to stdout and exit (for previewing layout)")
	phaseFlag := flag.String("phase", "work", "phase for snapshot: work|short|long")
	remaining := flag.Duration("remaining", 0, "override remaining time for snapshot (e.g. 12m34s)")
	running := flag.Bool("running", false, "mark running in snapshot")
	muted := flag.Bool("muted", false, "mark muted in snapshot")
	info := flag.Bool("info", false, "show info panel in snapshot")
	options := flag.Bool("options", false, "show options panel in snapshot")
	optionIdx := flag.Int("optidx", 0, "selected option index in snapshot")
	completed := flag.Int("completed", 0, "pomodoros completed in snapshot")
	width := flag.Int("width", 120, "viewport width for snapshot")
	height := flag.Int("height", 40, "viewport height for snapshot")
	flag.Parse()

	if *snapshot {
		m := initialModel()
		switch *phaseFlag {
		case "short":
			m.phase = phaseShortBreak
		case "long":
			m.phase = phaseLongBreak
		default:
			m.phase = phaseWork
		}
		m.remaining = m.phaseDuration(m.phase)
		if *remaining > 0 {
			m.remaining = *remaining
		}
		m.running = *running
		m.muted = *muted
		m.showInfo = *info
		m.showOptions = *options
		m.optionIdx = *optionIdx
		m.completed = *completed
		m.width = *width
		m.height = *height
		fmt.Print(m.View().Content)
		return
	}

	sound.Init(pingAudio, ".flac")
	defer sound.Cleanup()

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
