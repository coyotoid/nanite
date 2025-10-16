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
		if !app.conn.Scanner.Scan() {
			return "", app.conn.Scanner.Err()
		}
		str.Write(app.conn.Scanner.Bytes())
		str.WriteRune(' ')
	}

	return str.String(), nil
}

func (app *App) Send(data string) (num int, err error) {
	if _, err := fmt.Fprintf(app.conn, "SEND %s\n", data); err != nil {
		return 0, err
	}

	if !app.conn.Scanner.Scan() {
		return 0, app.conn.Scanner.Err()
	}
	numRaw := app.conn.Scanner.Text()
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

	if !app.conn.Scanner.Scan() {
		return 0, app.conn.Scanner.Err()
	}
	numRaw := app.conn.Scanner.Text()
	num, err = strconv.Atoi(numRaw)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (app *App) Skip(since int) (num int, err error) {
	if _, err := fmt.Fprintf(app.conn, "SKIP %d\n", since); err != nil {
		return 0, err
	}

	if !app.conn.Scanner.Scan() {
		return 0, app.conn.Scanner.Err()
	}
	num, err = strconv.Atoi(app.conn.Scanner.Text())
	if err != nil {
		return 0, err
	}

	for range num {
		if !app.conn.Scanner.Scan() {
			return 0, err
		}
		app.incoming <- MessageEvent(app.conn.Scanner.Text())
	}

	if !app.conn.Scanner.Scan() {
		return 0, app.conn.Scanner.Err()
	}
	last, err := strconv.Atoi(app.conn.Scanner.Text())
	if err != nil {
		return 0, err
	}
	app.incoming <- SetLastEvent(last)

	return num, nil
}

func (app *App) Last(n int) (num int, err error) {
	if n == 0 {
		return 0, nil
	}

	if _, err := fmt.Fprintf(app.conn, "LAST %d\n", n); err != nil {
		return 0, err
	}

	if !app.conn.Scanner.Scan() {
		return 0, app.conn.Scanner.Err()
	}
	numRaw := app.conn.Scanner.Text()
	num, err = strconv.Atoi(numRaw)
	if err != nil {
		return 0, err
	}

	for range num {
		if !app.conn.Scanner.Scan() {
			return 0, app.conn.Scanner.Err()
		}
		app.incoming <- MessageEvent(app.conn.Scanner.Text())
	}

	var last int
	if !app.conn.Scanner.Scan() {
		return 0, app.conn.Scanner.Err()
	}
	lastRaw := app.conn.Scanner.Text()
	last, err = strconv.Atoi(lastRaw)
	if err != nil {
		return 0, err
	}
	app.incoming <- SetLastEvent(last)

	return num, nil
}
