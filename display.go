package main

import (
	"fmt"
	"math"
	"strings"

	"codeberg.org/lobo/nanite/widgets/pager"
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
)

func (app *App) InitUI() error {
	var err error
	app.vx, err = vaxis.New(vaxis.Options{})
	if err != nil {
		return err
	}
	app.input = textinput.New()
	app.pager = &pager.Model{
		WrapMode: pager.WrapWords,
	}
	app.resize()
	return nil
}

func (app *App) FinishUI() {
	app.vx.Close()
}

func (app *App) resize() {
	win := app.vx.Window()
	app.w.log = win.New(0, 1, win.Width, win.Height-2)
	app.w.title = win.New(0, 0, win.Width, 1)
	app.w.input = win.New(0, win.Height-1, win.Width, 1)
	app.pager.Offset = math.MaxInt
}

func (app *App) Redraw() {
	app.w.title.Clear()

	titleStyle := vaxis.Style{Attribute: vaxis.AttrBold}
	delimiterStyle := vaxis.Style{Attribute: vaxis.AttrDim}

	if app.conn != nil {
		titleString := fmt.Sprintf("nanite (%s:%s)", app.host, app.port)
		app.vx.SetTitle(titleString)

		rateString := "manual"
		if app.rate != 0 {
			rateString = app.rate.String()
		}

		segments := []vaxis.Segment{
			{Text: "• "},
			{Text: app.host, Style: titleStyle},
			{Text: " │ ", Style: delimiterStyle},
			{Text: fmt.Sprintf("↻ %s", rateString)},
		}

		if app.stats != "" {
			segments = append(segments,
				vaxis.Segment{Text: " │ ", Style: delimiterStyle},
				vaxis.Segment{Text: app.stats},
			)
		}

		app.w.title.PrintTruncate(0, segments...)
		app.pager.Layout()
		app.pager.Draw(app.w.log)
		app.input.Draw(app.w.input)
	} else {
		app.vx.SetTitle("nanite (disconnected)")
		app.w.title.PrintTruncate(0,
			vaxis.Segment{Text: "✕ "},
			vaxis.Segment{Text: "disconnected", Style: titleStyle},
		)
	}
	app.vx.Render()
}

func (app *App) submitTextInput() {
	if len(app.input.Characters()) == 0 {
		return
	}

	if app.input.Characters()[0].Grapheme == "/" {
		name, rest, _ := strings.Cut(app.input.String()[1:], " ")
		if cmd, ok := CommandMap[name]; ok {
			cmd(app, rest)
		} else {
			app.AppendSystemMessage("unknown command \"%s\"", name)
		}
	} else {
		message := fmt.Sprintf("%s: %s", app.nick, app.input.String())
		app.AppendMessage(message)
		app.outgoing <- Message(message)
		app.outgoing <- Stat("")
	}

	app.input.SetContent("")
}

func (app *App) HandleTerminalEvent(ev vaxis.Event) {
	switch ev := ev.(type) {
	case vaxis.Key:
		switch ev.String() {
		case "Up":
			app.pager.ScrollUp()
		case "Down":
			app.pager.ScrollDown()
		case "Enter":
			app.submitTextInput()
		case "Ctrl+p":
			app.outgoing <- ManualPoll(app.last)
		case "Ctrl+l":
			app.Redraw()
			app.vx.Refresh()
		case "Ctrl+c":
			app.stop()
		}
	case vaxis.Mouse:
		switch ev.Button {
		case vaxis.MouseWheelUp:
			app.pager.ScrollUpN(2)
		case vaxis.MouseWheelDown:
			app.pager.ScrollDownN(2)
		}
	case vaxis.Resize:
		app.resize()
	}

	app.input.Update(ev)
}
