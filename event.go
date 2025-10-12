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
	app.incoming <- Last(num)

	return nil
}

type Poll int

func (p Poll) HandleOutgoing(app *App) error {
	num, err := app.Poll(int(p))
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
		app.outgoing <- Stat("")
	}
	return nil
}

type ManualPoll int

func (p ManualPoll) HandleOutgoing(app *App) error {
	num, err := app.Poll(int(p))
	if err != nil {
		return err
	}
	app.incoming <- ManualPoll(num)
	if num == 0 {
		return nil
	}

	num, err = app.Skip(app.last)
	if err != nil {
		return err
	}
	if num != 0 {
		app.outgoing <- Stat("")
	}
	return nil
}

func (p ManualPoll) HandleIncoming(app *App) {
	if int(p) == 0 {
		app.AppendSystemMessage("poll: no new messages")
	} else {
		app.AppendSystemMessage("poll: retrieving %d messages", p)
	}
}

type Last int

func (m Last) HandleIncoming(app *App) {
	app.last = int(m)
}

type Stat string

func (data Stat) HandleIncoming(app *App) {
	app.stats = string(data)
}
func (_ Stat) HandleOutgoing(app *App) error {
	res, err := app.Stat()
	if err != nil {
		return err
	}
	app.incoming <- Stat(res)
	return nil
}
