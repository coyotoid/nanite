package main

import (
	"fmt"
	"math"

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
	app.w.log = win.New(0, 1, win.Width, win.Height-3)
	app.w.title = win.New(0, 0, win.Width, 1)
	app.w.status = win.New(0, win.Height-2, win.Width, 1)
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
		app.vx.SetTitle(fmt.Sprintf("nanite (%s:%s)", app.host, app.port))
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

	// draw statusbar
	statusStyle := vaxis.Style{Attribute: vaxis.AttrReverse}
	app.w.status.Fill(vaxis.Cell{Style: statusStyle})
	app.w.status.PrintTruncate(0,
		vaxis.Segment{
			Text:  app.nick,
			Style: statusStyle,
		},
	)

	// let the widgets draw themselves
	app.pager.Layout()
	app.pager.Draw(app.w.log)
	app.input.Draw(app.w.input)
	app.vx.Render()
}
