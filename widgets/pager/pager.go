// This is a "fork" of Vaxis' pager widget that adds Unicode line breaking.
// See <https://git.sr.ht/~rockorager/vaxis/tree/v0.15.0/item/widgets/pager/pager.go>.

package pager

import (
	"strings"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/rivo/uniseg"
)

const (
	WrapFast = iota
	WrapWords
)

var defaultFill = vaxis.Character{Grapheme: " "}

type Model struct {
	Segments []vaxis.Segment
	lines    []*line
	Fill     vaxis.Cell
	Offset   int
	WrapMode int
	width    int
}

type line struct {
	characters []vaxis.Cell
}

func (l *line) append(t vaxis.Cell) {
	l.characters = append(l.characters, t)
}

func (m *Model) Draw(win vaxis.Window) {
	w, h := win.Size()
	if w != m.width {
		m.width = w
		m.Layout()
	}
	if len(m.lines)-m.Offset < h {
		m.Offset = len(m.lines) - h
	}
	if m.Offset < 0 {
		m.Offset = 0
	}
	if m.Fill.Grapheme == "" {
		m.Fill.Character = defaultFill
	}
	win.Fill(m.Fill)
	for row, l := range m.lines {
		if row < m.Offset {
			continue
		}
		if (row - m.Offset) >= h {
			return
		}
		col := 0
		for _, cell := range l.characters {
			win.SetCell(col, row-m.Offset, cell)
			col += cell.Width
		}
	}
}

func (m *Model) layoutFast() {
	l := &line{}
	col := 0
	for _, seg := range m.Segments {
		for _, char := range vaxis.Characters(seg.Text) {
			if strings.ContainsRune(char.Grapheme, '\n') {
				m.lines = append(m.lines, l)
				l = &line{}
				col = 0
				continue
			}
			cell := vaxis.Cell{
				Character: char,
				Style:     seg.Style,
			}
			l.append(cell)
			col += char.Width
			if col >= m.width {
				m.lines = append(m.lines, l)
				l = &line{}
				col = 0
			}
		}
	}
}

func (m *Model) layoutSlow() {
	cols := m.width
	col := 0
	l := &line{}

	var (
		state   = -1
		segment string
	)

	for _, seg := range m.Segments {
		rest := seg.Text
		for len(rest) > 0 {
			segment, rest, _, state = uniseg.FirstLineSegmentInString(rest, state)
			chars := vaxis.Characters(segment)
			total := 0
			for _, char := range chars {
				total += char.Width
			}

			if total > cols || total+col > cols {
				m.lines = append(m.lines, l)
				l = &line{}
				col = 0
			}

			for _, char := range chars {
				if uniseg.HasTrailingLineBreakInString(char.Grapheme) && col != 0 {
					m.lines = append(m.lines, l)
					l = &line{}
					col = 0
					continue
				}
				cell := vaxis.Cell{
					Character: char,
					Style:     seg.Style,
				}
				l.append(cell)
				col += char.Width
				if col >= cols {
					m.lines = append(m.lines, l)
					l = &line{}
					col = 0
				}
			}
		}
	}
}

func (m *Model) Layout() {
	m.lines = []*line{}
	switch m.WrapMode {
	case WrapFast:
		m.layoutFast()
	case WrapWords:
		m.layoutSlow()
	}
}

// Scrolls the pager down n lines, if it can
func (m *Model) ScrollDown() {
	m.Offset += 1
}

func (m *Model) ScrollUp() {
	m.Offset -= 1
}

func (m *Model) ScrollDownN(n int) {
	m.Offset += n
}

func (m *Model) ScrollUpN(n int) {
	m.Offset -= n
}
