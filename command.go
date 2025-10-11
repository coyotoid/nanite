package main

import (
	"fmt"
	"strconv"
)

func (app *App) Send(data string) (num int, err error) {
	if _, err := fmt.Fprintf(app.conn, "SEND %s\n", data); err != nil {
		return 0, err
	}

	if !app.scanner.Scan() {
		return 0, app.scanner.Err()
	}
	numRaw := app.scanner.Text()
	num, err = strconv.Atoi(numRaw)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (app *App) Poll(since int) (num int, err error) {
	if _, err := fmt.Fprintf(app.conn, "POLL %d\n", since); err != nil {
		return 0, err
	}

	if !app.scanner.Scan() {
		return 0, app.scanner.Err()
	}
	numRaw := app.scanner.Text()
	num, err = strconv.Atoi(numRaw)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (app *App) Last(n int) (err error) {
	if _, err := fmt.Fprintf(app.conn, "LAST %d\n", n); err != nil {
		return err
	}

	var nsrv int
	if !app.scanner.Scan() {
		return app.scanner.Err()
	}
	nsrvRaw := app.scanner.Text()
	nsrv, err = strconv.Atoi(nsrvRaw)
	if err != nil {
		return err
	}

	for range nsrv {
		if !app.scanner.Scan() {
			return app.scanner.Err()
		}
		app.incoming <- Message(app.scanner.Text())
	}

	var last int
	if !app.scanner.Scan() {
		return app.scanner.Err()
	}
	lastRaw := app.scanner.Text()
	last, err = strconv.Atoi(lastRaw)
	if err != nil {
		return err
	}
	app.incoming <- SetLast(last)

	return nil
}
