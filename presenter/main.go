// k8s-present — Design A (Split Console) terminal presenter.
//
// Left pane: lesson content + command list (↑/↓ select, Enter runs one).
// Right pane: output of the last run command, streamed live.
//
// Commands run under a pty (creack/pty) so progress bars (curl, ctr pull)
// render the way they do in a real terminal. slides captures output through
// a pipe, which is why its progress bars never appeared.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

// ---------------------------------------------------------------------------
// Deck format (deck.md):
//
//	# Lesson title            <- lesson title (first # line)
//	## Slide title            <- new slide
//	> quote line              <- yellow quote
//	- bullet                  <- bullet
//	plain text                <- paragraph
//	```sh                     <- command block: every line = one command
//	curl ...
//	```
//	```text                   <- diagram: shown as-is, never executed
//	...
//	```
// ---------------------------------------------------------------------------

type Cmd struct {
	Text    string
	Done    bool
	HasExit bool
	Exit    int
}

type Line struct {
	Kind string // text, quote, bullet, diagram
	Text string
}

type Slide struct {
	Title string
	Lines []Line
	Cmds  []Cmd
}

type Deck struct {
	Title  string
	Slides []Slide
}

func parseDeck(path string) (Deck, error) {
	var deck Deck
	f, err := os.Open(path)
	if err != nil {
		return deck, err
	}
	defer f.Close()

	var cur *Slide
	inFence := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimRight(raw, "\r")

		if inFence != "" {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFence = ""
				continue
			}
			if cur == nil {
				continue
			}
			if inFence == "sh" {
				if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") {
					cur.Cmds = append(cur.Cmds, Cmd{Text: strings.TrimSpace(line)})
				}
			} else { // text diagram
				cur.Lines = append(cur.Lines, Line{Kind: "diagram", Text: line})
			}
			continue
		}

		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "```"):
			lang := strings.TrimSpace(strings.TrimPrefix(trim, "```"))
			if lang == "sh" || lang == "bash" || lang == "text" {
				inFence = lang
			} else {
				inFence = "text" // unknown fence: show, never run
			}
		case strings.HasPrefix(trim, "# "):
			if deck.Title == "" {
				deck.Title = strings.TrimSpace(strings.TrimPrefix(trim, "# "))
			}
		case strings.HasPrefix(trim, "## "):
			deck.Slides = append(deck.Slides, Slide{Title: strings.TrimSpace(strings.TrimPrefix(trim, "## "))})
			cur = &deck.Slides[len(deck.Slides)-1]
		case strings.HasPrefix(trim, "> "):
			if cur != nil {
				cur.Lines = append(cur.Lines, Line{Kind: "quote", Text: strings.TrimSpace(strings.TrimPrefix(trim, "> "))})
			}
		case strings.HasPrefix(trim, "- "):
			if cur != nil {
				cur.Lines = append(cur.Lines, Line{Kind: "bullet", Text: strings.TrimSpace(strings.TrimPrefix(trim, "- "))})
			}
		case trim == "" || trim == "---":
			// skip
		default:
			if cur != nil {
				cur.Lines = append(cur.Lines, Line{Kind: "text", Text: trim})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return deck, err
	}
	if len(deck.Slides) == 0 {
		return deck, fmt.Errorf("deck mein koi slide nahi mili (## headings chahiye): %s", path)
	}
	return deck, nil
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type startedMsg struct {
	cmd *exec.Cmd
	pty *os.File
}

type outputMsg struct {
	data []byte
}

type doneMsg struct {
	exit int
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type model struct {
	deck    Deck
	workdir string
	slide   int
	sel     int // selected command on this slide, -1 = none

	outCmd    string
	outStatus string // "", "running", "done (exit 0)", "exit 1"
	outLines  []string
	pending   string
	running   bool
	cmd       *exec.Cmd
	ptyFile   *os.File

	width, height int
	ready         bool
}

func (m model) curSlide() *Slide { return &m.deck.Slides[m.slide] }

func (m *model) setSlide(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(m.deck.Slides) {
		i = len(m.deck.Slides) - 1
	}
	m.slide = i
	m.sel = -1
	if len(m.curSlide().Cmds) > 0 {
		m.sel = 0
		// first not-done command
		for idx, c := range m.curSlide().Cmds {
			if !c.Done {
				m.sel = idx
				break
			}
		}
	}
}

func initialModel(deck Deck, workdir string) model {
	m := model{deck: deck, workdir: workdir, sel: -1}
	m.setSlide(0)
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) startCmd(c string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("bash", "-c", c)
		cmd.Dir = m.workdir
		cmd.Env = os.Environ()
		ptmx, err := pty.Start(cmd)
		if err != nil {
			return doneMsg{exit: 127}
		}
		return startedMsg{cmd: cmd, pty: ptmx}
	}
}

func readChunk(f *os.File) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := f.Read(buf)
		if n > 0 {
			return outputMsg{data: append([]byte(nil), buf[:n]...)}
		}
		if err != nil {
			return doneMsg{exit: -1} // EOF: exit code decided in Update
		}
		return doneMsg{exit: -1}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		if m.running && m.ptyFile != nil {
			_ = pty.Setsize(m.ptyFile, &pty.Winsize{Cols: uint16(max(20, m.outWidth())), Rows: uint16(max(5, m.outHeight()))})
		}
		return m, nil

	case tea.KeyMsg:
		if m.running {
			switch msg.Type {
			case tea.KeyCtrlC:
				if m.cmd != nil && m.cmd.Process != nil {
					_ = m.cmd.Process.Signal(os.Interrupt)
				}
				return m, nil
			}
			// command chal rahi hai: keys us ko forward karo (sudo password waghera).
			// Special keys ko un ke asal bytes mein bhejo, naam nahi.
			if m.ptyFile != nil {
				switch msg.Type {
				case tea.KeyRunes:
					_, _ = m.ptyFile.Write([]byte(string(msg.Runes)))
				case tea.KeyEnter:
					_, _ = m.ptyFile.Write([]byte("\r"))
				case tea.KeySpace:
					_, _ = m.ptyFile.Write([]byte(" "))
				case tea.KeyBackspace:
					_, _ = m.ptyFile.Write([]byte{0x7f})
				case tea.KeyTab:
					_, _ = m.ptyFile.Write([]byte("\t"))
				}
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.sel > 0 {
				m.sel--
			}
		case "down", "j":
			if m.sel >= 0 && m.sel < len(m.curSlide().Cmds)-1 {
				m.sel++
			}
		case "enter":
			if m.sel >= 0 && m.sel < len(m.curSlide().Cmds) {
				c := &m.curSlide().Cmds[m.sel]
				m.outCmd = c.Text
				m.outStatus = "running"
				m.outLines = nil
				m.pending = ""
				m.running = true
				return m, m.startCmd(c.Text)
			}
		case " ", "right", "l", "n":
			m.setSlide(m.slide + 1)
		case "left", "h", "p":
			m.setSlide(m.slide - 1)
		}
		return m, nil

	case startedMsg:
		m.cmd = msg.cmd
		m.ptyFile = msg.pty
		if m.width > 0 {
			_ = pty.Setsize(m.ptyFile, &pty.Winsize{Cols: uint16(max(20, m.outWidth())), Rows: uint16(max(5, m.outHeight()))})
		}
		return m, readChunk(m.ptyFile)

	case outputMsg:
		m.appendOutput(msg.data)
		if m.running && m.ptyFile != nil {
			return m, readChunk(m.ptyFile)
		}
		return m, nil

	case doneMsg:
		exit := msg.exit
		if m.cmd != nil {
			// pty EOF ke baad process ka wait karo taake asal exit code mile
			_ = m.cmd.Wait()
			if m.cmd.ProcessState != nil {
				exit = m.cmd.ProcessState.ExitCode()
			}
		}
		if m.ptyFile != nil {
			_ = m.ptyFile.Close()
			m.ptyFile = nil
		}
		m.running = false
		m.cmd = nil
		if m.pending != "" {
			m.outLines = append(m.outLines, m.pending)
			m.pending = ""
		}
		if m.sel >= 0 && m.sel < len(m.curSlide().Cmds) {
			c := &m.curSlide().Cmds[m.sel]
			c.Done = true
			c.HasExit = true
			c.Exit = exit
		}
		if exit == 0 {
			m.outStatus = "done (exit 0)"
		} else {
			m.outStatus = fmt.Sprintf("exit %d", exit)
		}
		// agle na-ki-hui command par jao
		for idx, c := range m.curSlide().Cmds {
			if !c.Done {
				m.sel = idx
				break
			}
		}
		return m, nil
	}
	return m, nil
}

// appendOutput: pty bytes ko lines mein toro. \r progress redraws ko alag
// lines bana deta hai (curl ki progress dikhti rehti hai).
func (m *model) appendOutput(data []byte) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	m.pending += s
	for {
		i := strings.Index(m.pending, "\n")
		if i < 0 {
			break
		}
		m.outLines = append(m.outLines, m.pending[:i])
		m.pending = m.pending[i+1:]
	}
	if len(m.outLines) > 1000 {
		m.outLines = m.outLines[len(m.outLines)-1000:]
	}
}

func (m model) outWidth() int {
	if m.width <= 0 {
		return 60
	}
	return m.width - m.leftWidth() - 4
}

func (m model) leftWidth() int {
	if m.width <= 0 {
		return 60
	}
	w := m.width * 55 / 100
	if w < 40 {
		w = 40
	}
	return w
}

func (m model) outHeight() int {
	if m.height <= 0 {
		return 15
	}
	h := m.height - 4
	if h < 5 {
		h = 5
	}
	return h
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))  // cyan
	quoteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("228"))            // yellow
	bulletStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46"))  // green
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("84"))             // green
	failStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))            // red
	headStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	outHeadOK   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("84"))
	outHeadBad  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	outHeadRun  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("228"))
)

func truncate(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 3 {
		return string(r[:w])
	}
	return string(r[:w-3]) + "..."
}

func (m model) View() string {
	if !m.ready {
		return "loading..."
	}
	var b strings.Builder

	// Header
	head := fmt.Sprintf("%s  ·  Slide %d/%d", m.deck.Title, m.slide+1, len(m.deck.Slides))
	b.WriteString(headStyle.Render(truncate(head, m.width)) + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, 120))) + "\n")

	leftW := m.leftWidth()
	bodyH := m.height - 4
	if bodyH < 5 {
		bodyH = 5
	}

	// ---- Left pane: slide content + commands
	var left strings.Builder
	s := m.curSlide()
	left.WriteString(titleStyle.Render(s.Title) + "\n\n")
	ww := wordwrap.NewWriter(leftW - 2)
	for _, ln := range s.Lines {
		switch ln.Kind {
		case "quote":
			ww.Write([]byte("› " + ln.Text + "\n"))
		case "bullet":
			ww.Write([]byte("• " + ln.Text + "\n"))
		case "diagram":
			ww.Write([]byte(ln.Text + "\n"))
		default:
			ww.Write([]byte(ln.Text + "\n"))
		}
	}
	_ = ww.Close()
	left.WriteString(ww.String())
	if len(s.Cmds) > 0 {
		left.WriteString("\n" + dimStyle.Render("Commands — ↑↓ select · Enter run:") + "\n")
		for i, c := range s.Cmds {
			mark := "  "
			text := truncate(c.Text, leftW-6)
			switch {
			case i == m.sel && m.running:
				mark = "▶ "
				left.WriteString(selStyle.Render(mark + text) + dimStyle.Render("  …running") + "\n")
			case i == m.sel:
				mark = "▶ "
				left.WriteString(selStyle.Render(mark+text) + "\n")
			case c.Done && c.HasExit && c.Exit == 0:
				left.WriteString(doneStyle.Render("✓ " + text) + "\n")
			case c.Done:
				left.WriteString(failStyle.Render("✗ " + text + fmt.Sprintf("  (exit %d)", c.Exit)) + "\n")
			default:
				left.WriteString("  " + text + "\n")
			}
		}
	} else {
		left.WriteString("\n" + dimStyle.Render("(is slide mein koi command nahi — Space se agla slide)") + "\n")
	}

	// ---- Right pane: output
	var right strings.Builder
	var outHead string
	switch {
	case m.outCmd == "":
		outHead = "Output — abhi koi command nahi chali"
	case m.outStatus == "running":
		outHead = "Output — $ " + m.outCmd + "  …running"
	case strings.HasPrefix(m.outStatus, "done"):
		outHead = "Output — $ " + m.outCmd + "  ✓ " + m.outStatus
	default:
		outHead = "Output — $ " + m.outCmd + "  ✗ " + m.outStatus
	}
	right.WriteString(outHeadRun.Render(truncate(outHead, m.outWidth())) + "\n")
	right.WriteString(dimStyle.Render(strings.Repeat("─", max(10, m.outWidth()))) + "\n")
	lines := m.outLines
	if m.pending != "" {
		lines = append(append([]string{}, lines...), m.pending)
	}
	avail := bodyH - 2
	if avail < 1 {
		avail = 1
	}
	if len(lines) > avail {
		lines = lines[len(lines)-avail:]
	}
	for _, ln := range lines {
		right.WriteString(truncate(ln, m.outWidth()) + "\n")
	}

	leftBox := lipgloss.NewStyle().Width(leftW).Height(bodyH).Render(left.String())
	rightBox := lipgloss.NewStyle().Width(m.outWidth()).Height(bodyH).
		Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).Render(right.String())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox) + "\n")

	// Footer
	keys := "↑↓ command select · Enter run · Space next slide · ← prev · q quit"
	if m.running {
		keys = "command chal rahi hai — Ctrl+C rokne ke liye"
	}
	b.WriteString(dimStyle.Render(truncate(keys, m.width)))
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	check := flag.Bool("check", false, "deck parse karke slides/commands gin lo, UI khole baghair")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: k8s-present [--check] <deck.md>")
		os.Exit(2)
	}
	deckPath, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	deck, err := parseDeck(deckPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "deck parse error:", err)
		os.Exit(1)
	}
	if *check {
		total := 0
		for i, s := range deck.Slides {
			fmt.Printf("slide %2d: %-55s commands=%d\n", i+1, truncate(s.Title, 55), len(s.Cmds))
			total += len(s.Cmds)
		}
		fmt.Printf("OK: %d slides, %d commands — %s\n", len(deck.Slides), total, deckPath)
		return
	}
	workdir := filepath.Dir(deckPath)
	p := tea.NewProgram(initialModel(deck, workdir), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
