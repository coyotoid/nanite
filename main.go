package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"codeberg.org/lobo/nanite/widgets/pager"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
)

type App struct {
	ctx  context.Context
	stop context.CancelFunc

	conn  *Conn
	stats string

	nick string
	last int

	incoming chan IncomingEvent
	outgoing chan OutgoingEvent
	error    chan error

	scripts map[string]string

	vx    *vaxis.Vaxis
	pager *pager.Model
	input *textinput.Model
	w     struct {
		log, title, input vaxis.Window
	}

	cfgHome string
}

type Conn struct {
	net.Conn
	*bufio.Scanner
	host, port string
	rate       time.Duration
	ticker     *time.Ticker
	ctx        context.Context
	stop       context.CancelFunc
}

func (app *App) AppendMessage(data string) {
	nick, _, found := strings.Cut(data, ": ")
	style := vaxis.Style{}
	if !found || strings.TrimSpace(nick) != nick {
		style.Attribute = vaxis.AttrItalic
	}
	app.pager.Segments = append(app.pager.Segments,
		vaxis.Segment{Text: data, Style: style},
		vaxis.Segment{Text: "\n"},
	)
	app.pager.Offset = math.MaxInt
}

func (app *App) AppendSystemMessage(format string, args ...any) {
	st := vaxis.Style{Attribute: vaxis.AttrDim}
	app.pager.Segments = append(app.pager.Segments,
		vaxis.Segment{Text: "* ", Style: st},
		vaxis.Segment{Text: fmt.Sprintf(format, args...), Style: st},
		vaxis.Segment{Text: "\n"},
	)
	app.pager.Offset = math.MaxInt
}

func (app *App) SetNick(nick string) {
	nick = strings.TrimSpace(nick)
	app.input.SetPrompt(fmt.Sprintf("[%s] ", nick))
	app.input.Prompt.Attribute = vaxis.AttrBold
	app.nick = nick
}

func (app *App) EnsureConfigDir() error {
	userCfg, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	app.cfgHome = path.Join(userCfg, "nanite")
	stat, err := os.Stat(app.cfgHome)
	if err != nil {
		if err := os.MkdirAll(app.cfgHome, 0o700); err != nil {
			return err
		}
	} else {
		if !stat.IsDir() {
			return fmt.Errorf("expected %s to be directory", app.cfgHome)
		}
	}
	return nil
}

func NewApp() *App {
	app := &App{}
	app.ctx, app.stop = context.WithCancel(context.Background())
	app.incoming = make(chan IncomingEvent, 256)
	app.outgoing = make(chan OutgoingEvent, 256)
	app.error = make(chan error)

	if err := app.EnsureConfigDir(); err != nil {
		panic(err)
	}

	if err := app.RefreshScripts(); err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case ev := <-app.outgoing:
				_, dial := ev.(DialEvent)
				if dial || app.conn != nil {
					if err := ev.HandleOutgoing(app); err != nil {
						app.error <- err
						return
					}
				} else {
					app.incoming <- SystemMessageEvent("not connected to any server")
				}
			case <-app.ctx.Done():
				return
			}
		}
	}()

	app.InitUI()

	return app
}

func (app *App) Loop() error {
	for {
		select {
		case ev := <-app.vx.Events():
			app.HandleTerminalEvent(ev)
		case ev := <-app.incoming:
			ev.HandleIncoming(app)
		case err := <-app.error:
			return err
		case <-app.ctx.Done():
			return nil
		}
		app.Redraw()
	}
}

func (app *App) Finish() {
	app.stop()
	app.FinishUI()
	if app.conn != nil {
		HangupEvent{}.HandleOutgoing(app)
	}
}

func init() {
	initCommandMap()
}

func main() {
	app := NewApp()
	defer app.Finish()

	args := os.Args
	if len(args) == 2 || len(args) == 3 {
		host := args[1]
		port := "44322"
		if len(args) == 3 {
			port = args[2]
		}
		app.outgoing <- DialEvent{host, port}
	}

	if data, err := os.ReadFile(path.Join(app.cfgHome, "nick")); err == nil {
		app.SetNick(string(data))
	} else {
		app.SetNick("someone")
	}

	app.AppendSystemMessage("welcome to nanite! :3")

	if err := app.Loop(); err != nil {
		app.FinishUI()
		panic(err)
	}
}
