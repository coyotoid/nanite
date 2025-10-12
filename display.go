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
	app.dirty = true

	win := app.vx.Window()
	app.w.log = win.New(0, 1, win.Width, win.Height-2)
	app.w.title = win.New(0, 0, win.Width, 1)
	app.w.input = win.New(0, win.Height-1, win.Width, 1)
	app.pager.Offset = math.MaxInt
}

func (app *App) Redraw() {
	if !app.dirty {
		return
	}
	app.dirty = false

	// set window title and draw titlebar
	app.w.title.Clear()
	titlebarStyle := vaxis.Style{Attribute: vaxis.AttrBold}

	if app.conn != nil {
		app.vx.SetTitle(fmt.Sprintf("%s:%s", app.host, app.port))
		app.w.title.PrintTruncate(0,
			vaxis.Segment{
				Text:  "• ",
				Style: titlebarStyle,
			},
			vaxis.Segment{
				Text:  app.host,
				Style: titlebarStyle,
			},
			vaxis.Segment{
				Text:  ":",
				Style: titlebarStyle,
			},
			vaxis.Segment{
				Text:  app.port,
				Style: titlebarStyle,
			},
		)
	} else {
		app.vx.SetTitle("nanite (disconnected)")
		app.w.title.PrintTruncate(0,
			vaxis.Segment{
				Text:  "✖ (disconnected)",
				Style: titlebarStyle,
			},
		)
	}

	// let the widgets draw themselves
	app.pager.Layout()
	app.pager.Draw(app.w.log)
	app.input.Draw(app.w.input)
	app.vx.Render()
}

func (app *App) HandleTerminalEvent(ev vaxis.Event) {
	app.dirty = true

	switch ev := ev.(type) {
	case vaxis.Mouse:
		switch ev.Button {
		case vaxis.MouseWheelUp:
			app.pager.ScrollUpN(2)
		case vaxis.MouseWheelDown:
			app.pager.ScrollDownN(2)
		}
	case vaxis.Resize:
		app.resize()
	case vaxis.Key:
		if ev.MatchString("ctrl+c") {
			app.stop()
		}
		switch {
		case ev.MatchString("up"):
			app.pager.ScrollUp()
		case ev.MatchString("down"):
			app.pager.ScrollDown()
		case ev.MatchString("ctrl+l"):
			app.Redraw()
			app.vx.Refresh()
			app.dirty = false
		case ev.MatchString("enter"):
			if len(app.input.Characters()) == 0 {
				break
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
			}

			app.input.SetContent("")
		}
	}

	app.input.Update(ev)
}
