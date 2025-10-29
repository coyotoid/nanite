# nanite

`nanite` is a terminal client for the [Nanochat] protocol.

Requires Go 1.24.4 or higher.  Build using `make build`.

> The upstream URL for this repository is <https://git.rhzm.org/lobo/nanite>.
> The repositories hosted on GitHub and Codeberg are mirrors.

# keybindings

- `Ctrl+C`: quit
- `Ctrl+L`: refresh screen
- `Ctrl+P`: poll
- Emacs-like bindings for text editing

# commands

- `/clear`: clear message log
- `/dial host [port]`: connect to server
- `/hangup`: disconnect from server
- `/help`: see command list
- `/me ...`: send IRC-style `/me` message
- `/nick [nickname]`: change nickname or see current nickname if no arguments given
- `/poll [time]`: set polling rate to time or poll manually if no arguments given
- `/quit`: self-explanatory (aliased to `/q`)
- `/script [name]`: run script or see script list if no arguments given (aliased to `/.`)
- `/send ...`: send raw message

[Nanochat]: https://git.phial.org/d6/nanochat
