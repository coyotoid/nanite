package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (app *App) RefreshScripts() error {
	app.scripts = make(map[string]string)

	scriptDir := path.Join(app.cfgHome, "scripts")
	if _, err := os.Stat(scriptDir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return filepath.Walk(scriptDir, func(p string, i os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case i.IsDir():
			return nil
		default:
			if data, err := os.ReadFile(p); err != nil {
				return err
			} else {
				app.scripts[path.Base(p)] = string(data)
				return nil
			}
		}
	})
}

func (app *App) LoadScript(name string) error {
	if script, ok := app.scripts[name]; ok {
		for line := range strings.Lines(script) {
			cmdName, rest, _ := strings.Cut(strings.TrimSpace(line), " ")
			if cmd, ok := CommandMap[cmdName]; ok {
				cmd(app, rest)
			} else {
				return fmt.Errorf("unknown command \"%s\"", cmdName)
			}
		}
	} else {
		return fmt.Errorf("not found")
	}
	return nil
}
