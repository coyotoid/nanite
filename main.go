package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"codeberg.org/lobo/nanite/widgets/pager"
	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/textinput"
)

var (
	ErrAlreadyConnected = errors.New("already connected")
)

type App struct {
	ctx  context.Context
	stop context.CancelFunc
	mu   sync.RWMutex

	conn       net.Conn
	scanner    *bufio.Scanner
	host, port string

	nick   string
	last   int
	ticker *time.Ticker

	incoming chan IncomingEvent
	outgoing chan OutgoingEvent

	vx *vaxis.Vaxis
	w  struct {
		log, title, status, input vaxis.Window
	}
	pager *pager.Model
	input *textinput.Model
	dirty bool
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
	app.incoming = make(chan IncomingEvent)
	app.outgoing = make(chan OutgoingEvent)
	app.ticker = time.NewTicker(2 * time.Second)

	go func() {
		for {
			select {
			case ev := <-app.outgoing:
				ev.HandleOutgoing(app)
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
}

func (app *App) AppendMessage(data string) {
	app.pager.Segments = append(app.pager.Segments,
		vaxis.Segment{Text: data},
		vaxis.Segment{Text: "\n"},
	)
	app.last += 1
	app.pager.Offset = math.MaxInt
	app.dirty = true
}

func (app *App) HandleTerminalEvent(ev vaxis.Event) {
	app.dirty = true

	switch ev := ev.(type) {
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
			if len(app.input.Characters()) != 0 {
				message := fmt.Sprintf("%s: %s", app.nick, app.input.String())
				app.AppendMessage(message)
				app.outgoing <- Message(message)
				app.input.SetContent("")
			}
		}
	}

	app.input.Update(ev)
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

	if err := app.Connect(args[1], port); err != nil {
		panic(err)
	}
	defer app.Disconnect()

	app.InitUI()
	defer app.FinishUI()

	app.nick = "wolfdog"
	go app.Last(20)

	for {
		select {
		case ev := <-app.vx.Events():
			app.HandleTerminalEvent(ev)
		case ev := <-app.incoming:
			ev.HandleIncoming(&app)
		case <-app.ticker.C:
			app.outgoing <- Poll(app.last)
		case <-app.ctx.Done():
			return
		}
		app.Redraw()
	}
}
