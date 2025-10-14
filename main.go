package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"codeberg.org/lobo/nanite/widgets/pager"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
)

var CommandMap map[string]func(*App, string)

func init() {
	CommandMap = map[string]func(*App, string){
		"help": func(app *App, rest string) {
			var s strings.Builder
			for name, _ := range CommandMap {
				if name == "q" {
					continue
				}
				s.WriteString(name)
				s.WriteRune(' ')
			}
			app.AppendSystemMessage("commands: %s", s.String())
		},
		"dial": func(app *App, rest string) {
			args := strings.Fields(rest)
			if len(args) < 1 || len(args) > 2 {
				app.AppendSystemMessage("usage: /connect host [port]")
			}
			host := args[0]
			port := "44322"
			if len(args) == 2 {
				port = args[1]
			}
			app.outgoing <- DialEvent{host, port}
		},
		"hangup": func(app *App, rest string) {
			app.outgoing <- HangupEvent{}
		},
		"nick": func(app *App, rest string) {
			if rest == "" {
				app.AppendSystemMessage("nick: your nickname is %s", app.nick)
			} else {
				app.SetNick(rest)
				app.AppendSystemMessage("nick: your nickname is now %s", app.nick)
				if err := os.WriteFile(path.Join(app.cfgHome, "nick"), []byte(rest), 0o700); err != nil {
					app.AppendSystemMessage("nick: failed to persist nickname: %s", err)
				}
			}
		},
		"poll": func(app *App, rest string) {
			if rest == "" {
				app.outgoing <- ManualPollEvent(app.last)
			} else {
				num, err := strconv.Atoi(rest)
				if err != nil {
					app.AppendSystemMessage("poll: invalid number %s", rest)
				} else {
					if num == 0 {
						app.conn.ticker.Stop()
						app.conn.rate = 0
						app.AppendSystemMessage("poll: disabled automatic polling")
					} else {
						app.conn.rate = time.Second * time.Duration(num)
						app.conn.ticker.Stop()
						app.conn.ticker = time.NewTicker(app.conn.rate)
						app.AppendSystemMessage("poll: polling every %s", app.conn.rate.String())
					}
				}
			}
		},
		"me": func(app *App, rest string) {
			msg := fmt.Sprintf("%s %s", app.nick, rest)
			app.AppendMessage(msg)
			app.outgoing <- MessageEvent(msg)
		},
		"clear": func(app *App, rest string) {
			app.pager.Segments = []vaxis.Segment{}
			app.pager.Layout()
			app.AppendSystemMessage("cleared message history")
		},
		"quit": func(app *App, rest string) {
			app.stop()
		},
		"q": func(app *App, rest string) {
			app.stop()
		},
	}
}

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

	vx *vaxis.Vaxis
	w  struct {
		log, title, input vaxis.Window
	}
	pager   *pager.Model
	input   *textinput.Model
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
	// TODO: make messages without a nick italic
	app.pager.Segments = append(app.pager.Segments,
		vaxis.Segment{Text: data},
		vaxis.Segment{Text: "\n"},
	)
	app.last += 1
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
	app.input.SetPrompt(fmt.Sprintf("%s: ", nick))
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
		if err := os.Mkdir(app.cfgHome, 0o700); err != nil {
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
		app.error <- err
	}

	go func() {
		for {
			select {
			case ev := <-app.outgoing:
				if err := ev.HandleOutgoing(app); err != nil {
					app.error <- err
					return
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
	HangupEvent{}.HandleOutgoing(app)
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
		panic(err)
	}
}
