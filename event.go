package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"
)

type IncomingEvent interface {
	HandleIncoming(*App)
}

type OutgoingEvent interface {
	HandleOutgoing(*App) error
}

type DialEvent struct {
	Host, Port string
}

func (ev DialEvent) HandleIncoming(app *App) {
	app.AppendSystemMessage("connected to %s:%s", ev.Host, ev.Port)
}

func (ev DialEvent) HandleOutgoing(app *App) error {
	var err error

	if app.conn != nil {
		app.incoming <- SystemMessageEvent(
			fmt.Sprintf("already connected to %s:%s", app.conn.host, app.conn.port),
		)
		return nil
	}
	conn, err := net.Dial("tcp", net.JoinHostPort(ev.Host, ev.Port))
	if err != nil {
		return err
	}
	ctx, stop := context.WithCancel(app.ctx)

	app.conn = &Conn{
		Conn:    conn,
		Scanner: bufio.NewScanner(conn),
		host:    ev.Host,
		port:    ev.Port,
		ctx:     ctx,
		stop:    stop,
	}

	// calculate latency
	latStart := time.Now()
	_, err = app.Poll(0)
	if err != nil {
		return err
	}
	delta := time.Since(latStart)
	delta = min(max(time.Second, delta*3/2), 5*time.Second).Round(time.Second)

	app.conn.rate = delta
	app.conn.ticker = time.NewTicker(delta)

	go func() {
		for {
			select {
			case <-app.conn.ticker.C:
				app.outgoing <- PollEvent(app.last)
			case <-app.conn.ctx.Done():
				return
			}
		}
	}()

	app.outgoing <- FetchEvent(50)
	app.outgoing <- StatEvent("")
	app.incoming <- ev

	return nil
}

type HangupEvent struct{ host, port string }

func (ev HangupEvent) HandleIncoming(app *App) {
	app.AppendSystemMessage("disconnected from %s:%s", ev.host, ev.port)
}

func (_ HangupEvent) HandleOutgoing(app *App) error {
	if app.conn == nil {
		app.incoming <- SystemMessageEvent("not connected to any server")
		return nil
	}

	host := app.conn.host
	port := app.conn.port

	app.conn.stop()
	app.conn.Write([]byte("QUIT\n"))
	app.conn.Close()
	app.conn.ticker.Stop()
	app.conn.ticker = nil
	app.conn = nil
	app.incoming <- HangupEvent{host, port}

	return nil
}

type FetchEvent int

func (ev FetchEvent) HandleOutgoing(app *App) error {
	_, err := app.Last(int(ev))
	return err
}

type MessageEvent string

func (ev MessageEvent) HandleIncoming(app *App) {
	app.AppendMessage(string(ev))

}
func (ev MessageEvent) HandleOutgoing(app *App) error {
	app.incoming <- ev
	num, err := app.Send(string(ev))
	if err != nil {
		return err
	}
	app.incoming <- SetLastEvent(num)

	return nil
}

type SystemMessageEvent string

func (ev SystemMessageEvent) HandleIncoming(app *App) {
	app.AppendSystemMessage("%s", string(ev))
}

type PollEvent int

func (ev PollEvent) HandleOutgoing(app *App) error {
	num, err := app.Poll(int(ev))
	if err != nil {
		return err
	}
	if num == 0 {
		return nil
	}

	num, err = app.Skip(app.last)
	if err != nil {
		return err
	}
	if num != 0 {
		app.outgoing <- StatEvent("")
	}
	return nil
}

type ManualPollEvent int

func (ev ManualPollEvent) HandleOutgoing(app *App) error {
	num, err := app.Poll(int(ev))
	if err != nil {
		return err
	}
	app.incoming <- ManualPollEvent(num)
	if num == 0 {
		return nil
	}

	num, err = app.Skip(app.last)
	if err != nil {
		return err
	}
	if num != 0 {
		app.outgoing <- StatEvent("")
	}
	return nil
}

func (ev ManualPollEvent) HandleIncoming(app *App) {
	if int(ev) == 0 {
		app.AppendSystemMessage("no new messages")
	} else {
		app.AppendSystemMessage("retrieving %d messages", ev)
	}
}

type SetLastEvent int

func (ev SetLastEvent) HandleIncoming(app *App) {
	app.last = int(ev)
}

type StatEvent string

func (ev StatEvent) HandleIncoming(app *App) {
	app.stats = string(ev)
}
func (_ StatEvent) HandleOutgoing(app *App) error {
	res, err := app.Stat()
	if err != nil {
		return err
	}
	app.incoming <- StatEvent(res)
	return nil
}
