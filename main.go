package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"codeberg.org/lobo/nanite/widgets/pager"
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

var CommandMap = map[string]func(*App, string){
	"nick": func(app *App, rest string) {
		args := strings.Fields(rest)
		switch len(args) {
		case 0:
			app.AppendSystemMessage("nick: your nickname is %s", app.nick)
		default:
			app.SetNick(args[0])
			app.AppendSystemMessage("nick: your nickname is now %s", app.nick)
		}
	},
	"poll": func(app *App, rest string) {
		if rest == "" {
			app.outgoing <- ManualPoll(app.last)
		} else {
			num, err := strconv.Atoi(rest)
			if err != nil {
				app.AppendSystemMessage("poll: invalid number %s", rest)
			} else {
				if num == 0 {
					app.ticker.Stop()
					app.rate = 0
					app.AppendSystemMessage("poll: disabled automatic polling")
				} else {
					app.rate = time.Second * time.Duration(num)
					app.ticker.Stop()
					app.ticker = time.NewTicker(app.rate)
					app.AppendSystemMessage("poll: polling every %s", app.rate.String())
				}
			}
		}
	},
	"me": func(app *App, rest string) {
		msg := fmt.Sprintf("%s %s", app.nick, rest)
		app.AppendMessage(msg)
		app.outgoing <- Message(msg)
	},
	"quit": func(app *App, rest string) {
		app.stop()
	},
	"q": func(app *App, rest string) {
		app.stop()
	},
}

type App struct {
	ctx  context.Context
	stop context.CancelFunc

	conn       net.Conn
	scanner    *bufio.Scanner
	host, port string
	stats      string

	nick   string
	last   int
	rate   time.Duration
	ticker *time.Ticker

	incoming chan IncomingEvent
	outgoing chan OutgoingEvent
	error    chan error

	vx *vaxis.Vaxis
	w  struct {
		log, title, status, input vaxis.Window
	}
	pager *pager.Model
	input *textinput.Model
}

func (app *App) Connect(host, port string) (err error) {
	if app.conn != nil {
		return ErrAlreadyConnected
	}
	app.host = host
	app.port = port
	app.conn, err = net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return err
	}
	app.scanner = bufio.NewScanner(app.conn)
	app.incoming = make(chan IncomingEvent, 256)
	app.outgoing = make(chan OutgoingEvent, 256)
	app.error = make(chan error)

	// Calculate latency
	now := time.Now()
	_, err = app.Poll(0)
	if err != nil {
		app.Disconnect()
		return err
	}
	delta := time.Since(now).Round(time.Second)
	delta = min(max(time.Second, delta*3/2), 5*time.Second)

	app.rate = delta
	app.ticker = time.NewTicker(delta)

	go func() {
		app.outgoing <- Stat("")
		app.Last(20)

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

	return nil
}

func (app *App) Disconnect() {
	if app.conn != nil {
		app.conn.Write([]byte("QUIT\n"))
		app.conn.Close()
		app.conn = nil
	}
	if app.ticker != nil {
		app.ticker.Stop()
		app.ticker = nil
	}
	if app.incoming != nil {
		close(app.incoming)
		app.incoming = nil
	}
	if app.outgoing != nil {
		close(app.outgoing)
		app.outgoing = nil
	}
}

func (app *App) AppendMessage(data string) {
	app.pager.Segments = append(app.pager.Segments,
		vaxis.Segment{Text: data},
		vaxis.Segment{Text: "\n"},
	)
	app.last += 1
	app.pager.Offset = math.MaxInt
}

func (app *App) AppendSystemMessage(format string, args ...any) {
	st := vaxis.Style{Foreground: vaxis.ColorGray, Attribute: vaxis.AttrDim}
	app.pager.Segments = append(app.pager.Segments,
		vaxis.Segment{Text: "* ", Style: st},
		vaxis.Segment{Text: fmt.Sprintf(format, args...), Style: st},
		vaxis.Segment{Text: "\n"},
	)
	app.pager.Offset = math.MaxInt
}

func (app *App) SetNick(nick string) {
	app.input.SetPrompt(fmt.Sprintf("%s: ", nick))
	app.input.Prompt.Attribute = vaxis.AttrBold
	app.nick = nick
}

func main() {
	args := os.Args
	if len(args) < 2 || len(args) > 3 {
		fmt.Printf("usage: %s host [port]\n", args[0])
		os.Exit(1)
	}

	port := "44322"
	if len(args) == 3 {
		port = args[2]
	}

	app := App{}
	app.ctx, app.stop = context.WithCancel(context.Background())
	defer app.stop()

	app.InitUI()
	app.Redraw()
	defer app.FinishUI()

	if err := app.Connect(args[1], port); err != nil {
		panic(err)
	}
	defer app.Disconnect()

	app.SetNick("wolfdog")

	for {
		select {
		case ev := <-app.vx.Events():
			app.HandleTerminalEvent(ev)
		case ev := <-app.incoming:
			ev.HandleIncoming(&app)
		case <-app.ticker.C:
			app.outgoing <- Poll(app.last)
		case err := <-app.error:
			app.FinishUI()
			panic(err)
		case <-app.ctx.Done():
			return
		}

		app.Redraw()
	}
}
