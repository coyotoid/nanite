package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Command func(*App, string)

var CommandMap map[string]Command

func initCommandMap() {
	CommandMap = map[string]Command{
		"help": func(app *App, rest string) {
			var s strings.Builder
			for name, _ := range CommandMap {
				if name == "q" {
					continue
				}
				s.WriteString(name)
				s.WriteRune(' ')
			}
			app.AppendSystemMessage("commands: %s", s.String())
		},
		"script": func(app *App, rest string) {
			if rest == "" {
				var s strings.Builder
				for _, script := range app.scripts {
					s.WriteString(script)
					s.WriteRune(' ')
				}
				app.AppendSystemMessage("scripts: %s", s.String())
			} else {
				if err := app.LoadScript(rest); err != nil {
					app.AppendSystemMessage("error loading script `%s`: %s", rest, err)
				}
			}
		},
		"send": func(app *App, rest string) {
			app.AppendMessage(rest)
			app.outgoing <- MessageEvent(rest)
		},
		"dial": func(app *App, rest string) {
			args := strings.Fields(rest)
			if len(args) < 1 || len(args) > 2 {
				app.AppendSystemMessage("usage: /connect host [port]")
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
				app.AppendSystemMessage("nick: your nickname is %s", app.nick)
			} else {
				app.SetNick(rest)
				app.AppendSystemMessage("nick: your nickname is now %s", app.nick)
				if err := os.WriteFile(path.Join(app.cfgHome, "nick"), []byte(rest), 0o600); err != nil {
					app.AppendSystemMessage("nick: failed to persist nickname: %s", err)
				}
			}
		},
		"poll": func(app *App, rest string) {
			if rest == "" {
				app.outgoing <- ManualPollEvent(app.last)
			} else {
				num, err := strconv.Atoi(rest)
				if err != nil {
					app.AppendSystemMessage("poll: invalid number %s", rest)
				} else {
					if num == 0 {
						app.conn.ticker.Stop()
						app.conn.rate = 0
						app.AppendSystemMessage("poll: disabled automatic polling")
					} else {
						app.conn.rate = time.Second * time.Duration(num)
						app.conn.ticker.Stop()
						app.conn.ticker = time.NewTicker(app.conn.rate)
						app.AppendSystemMessage("poll: polling every %s", app.conn.rate.String())
					}
				}
			}
		},
		"me": func(app *App, rest string) {
			msg := fmt.Sprintf("%s %s", app.nick, rest)
			app.AppendMessage(msg)
			app.outgoing <- MessageEvent(msg)
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

	// aliases
	CommandMap["q"] = CommandMap["quit"]
	CommandMap["."] = CommandMap["script"]
}
