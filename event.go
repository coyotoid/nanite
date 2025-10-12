package main

type IncomingEvent interface {
	HandleIncoming(*App)
}
type OutgoingEvent interface {
	HandleOutgoing(*App) error
}

type Message string

func (m Message) HandleIncoming(app *App) {
	app.AppendMessage(string(m))

}
func (m Message) HandleOutgoing(app *App) error {
	num, err := app.Send(string(m))
	if err != nil {
		return err
	}
	app.incoming <- SetLast(num)

	return nil
}

type Poll int

func (p Poll) HandleOutgoing(app *App) error {
	num, err := app.Poll(int(p))
	if err != nil {
		return err
	}
	err = app.Last(num)
	if err != nil {
		return err
	}
	return nil
}

type SetLast int

func (m SetLast) HandleIncoming(app *App) {
	app.last = int(m)
}
