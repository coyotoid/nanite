package main

import (
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Command func(*App, string)

var CommandMap map[string]Command
var AliasMap map[string]string

func initCommandMap() {
	CommandMap = map[string]Command{
		"help": func(app *App, rest string) {
			var s strings.Builder
			for _, name := range slices.Sorted(maps.Keys(CommandMap)) {
				s.WriteString(name)
				s.WriteRune(' ')
			}
			app.AppendSystemMessage("commands: %s", s.String())
		},
		"script": func(app *App, rest string) {
			if rest == "" {
				if err := app.RefreshScripts(); err != nil {
					app.AppendSystemMessage("failed to refresh scripts")
				}
				var s strings.Builder
				for _, script := range slices.Sorted(maps.Keys(app.scripts)) {
					s.WriteString(script)
					s.WriteRune(' ')
				}
				app.AppendSystemMessage("scripts: %s", s.String())
			} else {
				if err := app.LoadScript(rest); err != nil {
					app.AppendSystemMessage("error loading script \"%s\": %s", rest, err)
				}
			}
		},
		"send": func(app *App, rest string) {
			app.outgoing <- MessageEvent(rest)
		},
		"dial": func(app *App, rest string) {
			args := strings.Fields(rest)
			if len(args) < 1 || len(args) > 2 {
				app.AppendSystemMessage("usage: /dial host [port]")
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
				app.AppendSystemMessage("your nickname is %s", app.nick)
			} else {
				app.SetNick(rest)
				app.AppendSystemMessage("your nickname is now %s", app.nick)
				if err := os.WriteFile(path.Join(app.cfgHome, "nick"), []byte(rest), 0o600); err != nil {
					app.AppendSystemMessage("failed to persist nickname: %s", err)
				}
			}
		},
		"poll": func(app *App, rest string) {
			if rest == "" {
				app.outgoing <- ManualPollEvent(app.last)
			} else {
				if app.conn == nil {
					app.AppendSystemMessage("not connected to any server")
					return
				}
				num, err := strconv.Atoi(rest)
				if err != nil {
					app.AppendSystemMessage("invalid number \"%s\"", rest)
				} else {
					if num == 0 {
						app.conn.ticker.Stop()
						app.conn.rate = 0
						app.AppendSystemMessage("disabled automatic polling")
					} else {
						app.conn.rate = time.Second * time.Duration(num)
						app.conn.ticker.Stop()
						app.conn.ticker = time.NewTicker(app.conn.rate)
						app.AppendSystemMessage("polling rate set to %s", app.conn.rate.String())
					}
				}
			}
		},
		"me": func(app *App, rest string) {
			app.outgoing <- MessageEvent(fmt.Sprintf("%s %s", app.nick, rest))
		},
		"clear": func(app *App, rest string) {
			clear(app.pager.Segments)
			app.pager.Layout()
			app.AppendSystemMessage("cleared message history")
		},
		"quit": func(app *App, rest string) {
			app.stop()
		},
	}

	AliasMap = map[string]string {
		"q": "quit",
		".": "script",
	}
}
