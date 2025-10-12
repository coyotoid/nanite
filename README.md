# nanite

`nanite` is a terminal [Nanochat] client.

## build

```
$ go build .
```

## usage

```
$ ./nanite
usage: ./nanite host [port]
$ ./nanite very.real-server.com
```

keybindings:

- `Ctrl+C`: quit
- `Ctrl+L`: refresh screen
- `Ctrl+P`: poll

commands:

- `/q`, `/quit`: quit
- `/nick [nickname]`: change nick, if no arguments, show current nick
- `/me [is listening to music]`: IRC `/me` alike
- `/poll [n]`: change polling interval, if no arguments, poll manually

## won't support (yet)

- sixel (tried, it seems to be complicated to get it to work with Vaxis' pager
  widget)

[Nanochat]: https://git.phial.org/d6/nanochat
[Vaxis]: https://git.sr.ht/~rockorager/vaxis
