package main

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (app *App) RefreshScripts() error {
	scriptDir := path.Join(app.cfgHome, "scripts")
	if _, err := os.Stat(scriptDir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return filepath.Walk(scriptDir, func(p string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if i.IsDir() {
			return nil
		}
		app.scripts = append(app.scripts, path.Base(p))
		return nil
	})
}

func (app *App) LoadScript(name string) error {
	scriptPath := path.Join(app.cfgHome, "scripts", name)
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}

	for line := range strings.Lines(string(data)) {
		app.AppendSystemMessage("/%s", line)
		name, rest, _ := strings.Cut(line, " ")
		if cmd, ok := CommandMap[name]; ok {
			cmd(app, rest)
		} else {
			app.AppendSystemMessage("unknown command \"%s\"", name)
		}
	}

	return nil
}
