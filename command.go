package main

import (
	"fmt"
	"strconv"
	"strings"
)

func (app *App) Stat() (res string, err error) {
	if _, err := app.conn.Write([]byte("STAT\n")); err != nil {
		return "", err
	}

	var str strings.Builder
	for range 3 {
		if !app.scanner.Scan() {
			return "", app.scanner.Err()
		}
		str.Write(app.scanner.Bytes())
		str.WriteRune(' ')
	}

	return str.String(), nil
}

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

func (app *App) Last(n int) (num int, err error) {
	if n == 0 {
		return 0, nil
	}

	if _, err := fmt.Fprintf(app.conn, "LAST %d\n", n); err != nil {
		return 0, err
	}

	var nsrv int
	if !app.scanner.Scan() {
		return 0, app.scanner.Err()
	}
	nsrvRaw := app.scanner.Text()
	nsrv, err = strconv.Atoi(nsrvRaw)
	if err != nil {
		return 0, err
	}

	if nsrv != 0 {
		for range nsrv {
			if !app.scanner.Scan() {
				return 0, app.scanner.Err()
			}
			app.incoming <- Message(app.scanner.Text())
		}
	}

	var last int
	if !app.scanner.Scan() {
		return 0, app.scanner.Err()
	}
	lastRaw := app.scanner.Text()
	last, err = strconv.Atoi(lastRaw)
	if err != nil {
		return 0, err
	}
	app.incoming <- Last(last)

	return nsrv, nil
}
