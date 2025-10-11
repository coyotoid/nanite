package main

type IncomingEvent interface {
	HandleIncoming(*App)
}
type OutgoingEvent interface {
	HandleOutgoing(*App)
}

type Message string

func (m Message) HandleIncoming(app *App) {
	app.AppendMessage(string(m))
}
func (m Message) HandleOutgoing(app *App) {
	num, _ := app.Send(string(m))
	app.incoming <- SetLast(num)

}

type Poll int

func (p Poll) HandleOutgoing(app *App) {
	num, _ := app.Poll(int(p))
	go app.Last(num)
}

type SetLast int

func (m SetLast) HandleIncoming(app *App) {
	app.last = int(m)
}
