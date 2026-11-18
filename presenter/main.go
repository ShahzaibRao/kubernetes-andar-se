// k8s-present v2 — approved Design v2 (Split Console, static left panel).
//
// Layout:
//
//	Header  — lesson title + Slide N/M (static)
//	Left    — slide content (Kya/Kyun/Kaise + HINT) + command list.
//	          PINNED: rendered once per slide/size/selection and cached, so
//	          streaming output can never move, flicker or hide it.
//	Right   — Terminal: one output BLOCK per command (persistent history,
//	          `$ cmd` header + status chip), live progress inside the block,
//	          plus a free input line at the bottom (Tab or / to type your
//	          own command, Enter runs it, Esc back to the list).
//
// v1 bugs fixed here:
//  1. Right box was 1 column too wide (border+padding not counted) so the
//     whole render wrapped on narrow terminals — that is what made the
//     left pane "fluctuate and disappear" during image pull.
//  2. Child pty output carried raw ANSI escapes into the TUI render and
//     corrupted the screen (clear-line / cursor moves). Output is now
//     sanitized before it is stored or drawn.
//  3. \r turned every progress redraw into a new line and flooded the
//     pane. \r now overwrites the current line, like a real terminal.
//  4. Only the last command's output was kept. Blocks persist for the
//     whole session, across slides.
//  5. Output chunks re-rendered the screen dozens of times a second.
//     Chunks are batched and flushed at most every 80ms.
//
// Commands run under a pty (creack/pty) so progress bars (curl, ctr pull)
// render the way they do in a real terminal.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	"github.com/muesli/reflow/wordwrap"
)

// ---------------------------------------------------------------------------
// Deck format (deck.md):
//
//	# Lesson title            <- lesson title (first # line)
//	## Slide title            <- new slide (takeaway sentence, not a label)
//	> quote line              <- HINT box: the say-line, WHY in one sentence
//	! expect line             <- EXPECT box: what success looks like
//	- bullet                  <- bullet (supports **bold**)
//	plain text                <- paragraph (supports **bold**)
//	```sh                     <- command block: every line = one command
//	# comment line            <- annotation, shown dim above the next command
//	curl ...
//	```
//	```text                   <- diagram: shown as-is, never executed
//	...
//	```
// ---------------------------------------------------------------------------

type Cmd struct {
	Text    string
	Note    string // # comment line above it in the deck (shown dim, not run)
	Done    bool
	HasExit bool
	Exit    int
}

type Line struct {
	Kind string // text, quote, expect, bullet, diagram
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
	pendingNote := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimRight(raw, "\r")

		if inFence != "" {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFence = ""
				pendingNote = ""
				continue
			}
			if cur == nil {
				continue
			}
			if inFence == "sh" {
				t := strings.TrimSpace(line)
				switch {
				case t == "":
					// skip
				case strings.HasPrefix(t, "#"):
					note := strings.TrimSpace(strings.TrimPrefix(t, "#"))
					if pendingNote == "" {
						pendingNote = note
					} else {
						pendingNote += " / " + note
					}
				default:
					cur.Cmds = append(cur.Cmds, Cmd{Text: strings.TrimSpace(line), Note: pendingNote})
					pendingNote = ""
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
		case strings.HasPrefix(trim, "! "):
			if cur != nil {
				cur.Lines = append(cur.Lines, Line{Kind: "expect", Text: strings.TrimSpace(strings.TrimPrefix(trim, "! "))})
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
// Output blocks — one per command run, kept for the whole session.
// ---------------------------------------------------------------------------

type Block struct {
	Cmd      string
	Custom   bool // typed in the free input line, not a slide command
	SlideIdx int
	CmdIdx   int // -1 for custom commands
	Lines    []string
	Cur      string // partial line still being written
	CurOver  bool   // Cur overwrites the last line (after \r)
	Running  bool
	Done     bool
	Exit     int
}

// displayLines: committed lines + the in-progress line (in place when it is
// an overwrite of the last line, e.g. a live progress bar).
func (b *Block) displayLines() []string {
	if b.Cur == "" {
		return b.Lines
	}
	if b.CurOver && len(b.Lines) > 0 {
		out := append([]string{}, b.Lines...)
		out[len(out)-1] = b.Cur
		return out
	}
	return append(append([]string{}, b.Lines...), b.Cur)
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

type flushMsg struct{}

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

	blocks      []Block
	curBlockIdx int // -1 = none running
	running     bool
	cmd         *exec.Cmd
	ptyFile     *os.File

	// batched pty output: chunks accumulate here, flushed every 80ms
	rawPending   []byte
	flushPending bool

	// free terminal input
	input      string
	focusInput bool

	// cached left pane (invalidated on slide/selection/size/done change)
	leftCache    string
	leftCacheKey string

	width, height int
	ready         bool
}

func (m model) curSlide() *Slide { return &m.deck.Slides[m.slide] }

func (m *model) curBlock() *Block {
	if m.curBlockIdx < 0 || m.curBlockIdx >= len(m.blocks) {
		return nil
	}
	return &m.blocks[m.curBlockIdx]
}

func (m *model) setSlide(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(m.deck.Slides) {
		i = len(m.deck.Slides) - 1
	}
	m.slide = i
	m.sel = -1
	m.focusInput = false
	if len(m.curSlide().Cmds) > 0 {
		m.sel = 0
		for idx, c := range m.curSlide().Cmds {
			if !c.Done {
				m.sel = idx
				break
			}
		}
	}
	m.leftCache = ""
}

func initialModel(deck Deck, workdir string) model {
	m := model{deck: deck, workdir: workdir, sel: -1, curBlockIdx: -1}
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

func flushTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return flushMsg{} })
}

// ---------------------------------------------------------------------------
// Output sanitizing + \r handling
// ---------------------------------------------------------------------------

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]|\x1b\\][^\x07]*\x07|\x1b[()][0-9A-Z]")

// control chars except \n \r \t — never let them reach the screen
var ctrlRe = regexp.MustCompile("[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]")

func stripANSI(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	s = ctrlRe.ReplaceAllString(s, "")
	return s
}

// commitCur: the partial line in Cur becomes a display line.
// isCR=true (after \r): the NEXT content overwrites this line.
// isCR=false (after \n): fresh line next.
func (m *model) commitCur(isCR bool) {
	b := m.curBlock()
	if b == nil {
		return
	}
	if b.Cur == "" {
		b.CurOver = isCR
		return
	}
	if b.CurOver && len(b.Lines) > 0 {
		b.Lines[len(b.Lines)-1] = b.Cur
	} else {
		b.Lines = append(b.Lines, b.Cur)
	}
	b.Cur = ""
	b.CurOver = isCR
}

// feedBytes: sanitized pty bytes -> current block, \r overwrites like a
// real terminal instead of flooding the pane with progress lines.
func (m *model) feedBytes(data []byte) {
	b := m.curBlock()
	if b == nil || len(data) == 0 {
		return
	}
	s := stripANSI(string(data))
	for _, r := range s {
		switch r {
		case '\r':
			m.commitCur(true)
		case '\n':
			m.commitCur(false)
		default:
			b.Cur += string(r)
		}
	}
	m.trimBlocks()
}

// trimBlocks: keep at most 2000 lines of history; drop oldest finished
// blocks first, never the running one or the last block.
func (m *model) trimBlocks() {
	total := 0
	for i := range m.blocks {
		total += len(m.blocks[i].displayLines()) + 1
	}
	for total > 2000 && len(m.blocks) > 1 {
		dropped := false
		for i := range m.blocks {
			if m.blocks[i].Done && i != m.curBlockIdx {
				total -= len(m.blocks[i].displayLines()) + 1
				m.blocks = append(m.blocks[:i], m.blocks[i+1:]...)
				if m.curBlockIdx > i {
					m.curBlockIdx--
				}
				dropped = true
				break
			}
		}
		if !dropped {
			break
		}
	}
}

func (m *model) runCommand(text string, slideIdx, cmdIdx int, custom bool) tea.Cmd {
	m.blocks = append(m.blocks, Block{
		Cmd: text, Custom: custom, SlideIdx: slideIdx, CmdIdx: cmdIdx, Running: true,
	})
	m.curBlockIdx = len(m.blocks) - 1
	m.running = true
	m.focusInput = false
	return m.startCmd(text)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.leftCache = ""
		if m.running && m.ptyFile != nil {
			_ = pty.Setsize(m.ptyFile, &pty.Winsize{Cols: uint16(max(20, m.textWidth())), Rows: uint16(max(5, m.termRows()))})
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

		if m.focusInput {
			switch msg.Type {
			case tea.KeyEsc:
				m.focusInput = false
				m.leftCache = ""
				return m, nil
			case tea.KeyTab:
				m.focusInput = false
				m.leftCache = ""
				return m, nil
			case tea.KeyEnter:
				if strings.TrimSpace(m.input) == "" {
					return m, nil
				}
				text := m.input
				m.input = ""
				return m, m.runCommand(text, m.slide, -1, true)
			case tea.KeyBackspace:
				if m.input != "" {
					r := []rune(m.input)
					m.input = string(r[:len(r)-1])
				}
				return m, nil
			case tea.KeyCtrlU:
				m.input = ""
				return m, nil
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyRunes:
				m.input += string(msg.Runes)
				return m, nil
			case tea.KeySpace:
				m.input += " "
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.sel > 0 {
				m.sel--
				m.leftCache = ""
			}
		case "down", "j":
			if m.sel >= 0 && m.sel < len(m.curSlide().Cmds)-1 {
				m.sel++
				m.leftCache = ""
			}
		case "enter":
			if m.sel >= 0 && m.sel < len(m.curSlide().Cmds) {
				c := m.curSlide().Cmds[m.sel]
				return m, m.runCommand(c.Text, m.slide, m.sel, false)
			}
		case "tab", "/":
			m.focusInput = true
			m.leftCache = ""
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
			_ = pty.Setsize(m.ptyFile, &pty.Winsize{Cols: uint16(max(20, m.textWidth())), Rows: uint16(max(5, m.termRows()))})
		}
		return m, readChunk(m.ptyFile)

	case outputMsg:
		m.rawPending = append(m.rawPending, msg.data...)
		cmd := tea.Cmd(nil)
		if m.running && m.ptyFile != nil {
			cmd = readChunk(m.ptyFile)
		}
		if !m.flushPending {
			m.flushPending = true
			if cmd != nil {
				return m, tea.Batch(cmd, flushTick())
			}
			return m, flushTick()
		}
		return m, cmd

	case flushMsg:
		m.flushPending = false
		if len(m.rawPending) > 0 {
			m.feedBytes(m.rawPending)
			m.rawPending = nil
		}
		return m, nil

	case doneMsg:
		// bacha hua output pehle flush karo
		if len(m.rawPending) > 0 {
			m.feedBytes(m.rawPending)
			m.rawPending = nil
		}
		m.flushPending = false
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
		b := m.curBlock()
		if b != nil {
			if b.Cur != "" {
				m.commitCur(false)
			}
			b.Running = false
			b.Done = true
			b.Exit = exit
			if b.CmdIdx >= 0 && b.SlideIdx == m.slide && b.CmdIdx < len(m.curSlide().Cmds) {
				c := &m.curSlide().Cmds[b.CmdIdx]
				c.Done = true
				c.HasExit = true
				c.Exit = exit
			}
		}
		m.running = false
		m.cmd = nil
		m.curBlockIdx = -1
		m.leftCache = ""
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

// ---------------------------------------------------------------------------
// Sizing — border + padding + separator counted (v1 bug: 1 column too wide)
// ---------------------------------------------------------------------------

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

// terminal pane content width: total - left - separator(1) - border(2) - padding(2)
func (m model) outWidth() int {
	if m.width <= 0 {
		return 60
	}
	w := m.width - m.leftWidth() - 5
	if w < 20 {
		w = 20
	}
	return w
}

// textWidth: asal likhne ki jagah. lipgloss Width border/padding ko andar
// gin sakta hai, is liye text ko 4 col chhota rakho warna lambi header
// wrap ho jati hai (v2 test mein dekha: "https..." doosri line par aa gaya tha).
func (m model) textWidth() int {
	w := m.outWidth() - 4
	if w < 20 {
		w = 20
	}
	return w
}

func (m model) bodyHeight() int {
	if m.height <= 0 {
		return 15
	}
	h := m.height - 4 // header 2 + footer 1 + spare 1
	if h < 5 {
		h = 5
	}
	return h
}

// terminal pane ke andar blocks ke liye lines: border(2) + title(1) + sep(1) + input(1)
func (m model) termRows() int {
	r := m.bodyHeight() - 5
	if r < 1 {
		r = 1
	}
	return r
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51")) // cyan
	quoteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("228"))           // yellow
	expectStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("84")) // green
	bulletStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46")) // green
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("84"))            // green
	failStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))           // red
	headStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	termStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
	inputStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46"))
)

// renderInline: **bold** segments ko bold karo, baqi base style mein.
// wordwrap ANSI-aware hai, is liye style pehle lagana theek hai.
func renderInline(s string, base lipgloss.Style) string {
	parts := strings.Split(s, "**")
	var sb strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			sb.WriteString(base.Bold(true).Render(p))
		} else {
			sb.WriteString(base.Render(p))
		}
	}
	return sb.String()
}

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

func (m model) leftSignature() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d|%d|%d|%d|%d|%v", m.slide, m.sel, m.width, m.bodyHeight(), len(m.curSlide().Cmds), m.running)
	for _, c := range m.curSlide().Cmds {
		if c.Done {
			fmt.Fprintf(&sb, "|%d", c.Exit)
		} else {
			sb.WriteString("|.")
		}
	}
	return sb.String()
}

// renderLeft: slide content + commands. Result is cached — output streaming
// never re-renders it, so the left pane cannot flicker or disappear.
func (m *model) renderLeft() string {
	key := m.leftSignature()
	if m.leftCache != "" && key == m.leftCacheKey {
		return m.leftCache
	}
	m.leftCacheKey = key

	leftW := m.leftWidth()
	var sb strings.Builder
	s := m.curSlide()
	sb.WriteString(titleStyle.Render(s.Title) + "\n\n")
	ww := wordwrap.NewWriter(leftW - 2)
	for _, ln := range s.Lines {
		switch ln.Kind {
		case "quote":
			ww.Write([]byte(renderInline("HINT › "+ln.Text, quoteStyle) + "\n"))
		case "expect":
			ww.Write([]byte(renderInline("EXPECT › "+ln.Text, expectStyle) + "\n"))
		case "bullet":
			ww.Write([]byte(renderInline("• "+ln.Text, bulletStyle) + "\n"))
		case "diagram":
			ww.Write([]byte(dimStyle.Render(ln.Text) + "\n"))
		default:
			ww.Write([]byte(renderInline(ln.Text, bulletStyle) + "\n"))
		}
	}
	_ = ww.Close()
	sb.WriteString(ww.String())
	if len(s.Cmds) > 0 {
		sb.WriteString("\n" + dimStyle.Render("Commands — ↑↓ select · Enter run:") + "\n")
		for i, c := range s.Cmds {
			if c.Note != "" {
				sb.WriteString(dimStyle.Render("  # "+truncate(c.Note, leftW-6)) + "\n")
			}
			text := truncate(c.Text, leftW-6)
			runningHere := m.running && m.curBlock() != nil && m.curBlock().SlideIdx == m.slide && m.curBlock().CmdIdx == i
			switch {
			case runningHere:
				sb.WriteString(selStyle.Render("▶ "+text) + dimStyle.Render("  …running") + "\n")
			case i == m.sel:
				sb.WriteString(selStyle.Render("▶ "+text) + "\n")
			case c.Done && c.HasExit && c.Exit == 0:
				sb.WriteString(doneStyle.Render("✓ "+text) + "\n")
			case c.Done:
				sb.WriteString(failStyle.Render(fmt.Sprintf("✗ %s  (exit %d)", text, c.Exit)) + "\n")
			default:
				sb.WriteString("  " + text + "\n")
			}
		}
	} else {
		sb.WriteString("\n" + dimStyle.Render("(is slide mein koi command nahi — Space se agla slide)") + "\n")
	}
	m.leftCache = sb.String()
	return m.leftCache
}

// renderTerminal: persistent output blocks + free input line.
func (m model) renderTerminal() string {
	var sb strings.Builder
	sb.WriteString(termStyle.Render("Terminal") + "\n")
	sb.WriteString(dimStyle.Render(strings.Repeat("─", max(10, m.outWidth()))) + "\n")

	avail := m.termRows() // title + sep + input already counted in termRows
	if avail < 1 {
		avail = 1
	}
	// flat render lines: block header + its output lines
	var flat []string
	for _, b := range m.blocks {
		var head string
		prefix := "$ "
		if b.Custom {
			prefix = "› $ "
		}
		// header ko bhi truncate karo warna lamba command box mein wrap ho jata hai
		switch {
		case b.Running:
			head = outHeadRun.Render(truncate(prefix+b.Cmd+"  …running", m.textWidth()))
		case b.Exit == 0:
			head = outHeadOK.Render(truncate(prefix+b.Cmd+"  ✓ exit 0", m.textWidth()))
		default:
			head = outHeadBad.Render(truncate(fmt.Sprintf("%s%s  ✗ exit %d", prefix, b.Cmd, b.Exit), m.textWidth()))
		}
		flat = append(flat, head)
		for _, ln := range b.displayLines() {
			flat = append(flat, truncate(ln, m.textWidth()))
		}
	}
	if len(m.blocks) == 0 {
		flat = append(flat, dimStyle.Render("(abhi koi command nahi chali — list se Enter, ya Tab dabake apni command likho)"))
	}
	if len(flat) > avail {
		flat = flat[len(flat)-avail:]
	}
	for _, ln := range flat {
		sb.WriteString(ln + "\n")
	}

	// free input line (always visible at the bottom)
	if m.focusInput {
		sb.WriteString(inputStyle.Render("› " + m.input + "▊"))
	} else if m.running {
		sb.WriteString(dimStyle.Render("› command chal rahi hai…"))
	} else {
		sb.WriteString(dimStyle.Render("› Tab ya / dabao — apni command likho"))
	}
	return sb.String()
}

var (
	outHeadOK  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("84"))
	outHeadBad = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	outHeadRun = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("228"))
)

func (m model) View() string {
	if !m.ready {
		return "loading..."
	}
	if m.width < 70 || m.height < 14 {
		return "Terminal chhota hai — kam se kam 80x24 karo, phir dobara chalao.\n"
	}
	var b strings.Builder

	// Header (static) — series branding + lesson + slide number
	head := fmt.Sprintf("Kubernetes Andar Se  ·  %s  ·  Slide %d/%d", m.deck.Title, m.slide+1, len(m.deck.Slides))
	b.WriteString(headStyle.Render(truncate(head, m.width)) + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, 120))) + "\n")

	bodyH := m.bodyHeight()
	leftW := m.leftWidth()

	leftBox := lipgloss.NewStyle().Width(leftW).Height(bodyH).Render(m.renderLeft())
	// Height content ki hoti hai — border 2 lines aur add karta hai, is liye
	// right box ki content height bodyH-2 hai taake total bodyH rahe.
	rightBox := lipgloss.NewStyle().Width(m.outWidth()).Height(bodyH-2).
		Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).Render(m.renderTerminal())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox) + "\n")

	// Footer
	var keys string
	switch {
	case m.running:
		keys = "command chal rahi hai — Ctrl+C rokne ke liye"
	case m.focusInput:
		keys = "Enter: chalao · Esc: wapas list par · Ctrl+U: saaf karo"
	default:
		keys = "↑↓ command select · Enter run · Tab apni command · Space agla slide · ← pichla · q quit"
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
