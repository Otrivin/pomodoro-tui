# Pomodoro TUI

Super simple terminal based Pomodoro timer built with built with
[Bubble Tea][bubbletea] and [Lipgloss][lipgloss].

[!Pomodoro](assets/pomodoro_gui.png)

## What is it?

It's trying to mimic the pomodoro technique, as listed on Wikipedia

*The original technique has six steps:

    Decide on the task to be done.
    Set the Pomodoro timer (typically for 25 minutes).[1]
    Work on the task.
    End work when the timer rings and take a short break (typically 5–10 minutes).[5]
    Go back to Step 2 and repeat until you complete four pomodori.
    After four pomodori are done, take a long break (typically 20 to 30 minutes) instead of a short break. Once the long break is finished, return to step 2.

For the purposes of the technique, a pomodoro is an interval of work time (and pomodori is the plural form).*

## Ok, I wanna install it. Pretty please?

### Download the binary

Grab the archive for your platform from the [Releases][releases] page,
extract, and run the `pomodoro` binary. No installation step required.

### Source it like a trooper!

Requires Go 1.25 or newer.

On **Linux**, building also needs the ALSA development headers (used by the
embedded audio stack to link against `libasound`):

```bash
# Debian / Ubuntu
sudo apt install libasound2-dev
# Fedora / RHEL
sudo dnf install alsa-lib-devel
# Arch
sudo pacman -S alsa-lib
```

macOS and Windows builds need no extra system packages.

```bash
git clone https://github.com/Otrivin/pomodoro-tui.git
cd pomodoro-tui
go build -ldflags="-s -w" -trimpath -o pomodoro .
./pomodoro
```

## Runtime requirements

None beyond the OS's native audio stack. The audio file is embedded in the binary, decoded in-process and played through:

- Linux → `libasound.so.2` (ALSA; present on every Linux desktop)
- macOS → CoreAudio (ships with macOS)
- Windows → WASAPI (ships with Windows)

If audio initialisation fails for any reason, the terminal bell (`\a`) is emitted as a fallback.

## Control it like Dr.Dre

| Key           | Action                        |
|---------------|-------------------------------|
| `space`/`enter` | Start / pause the timer     |
| `r`           | Reset current phase           |
| `s`           | Skip to next phase            |
| `n`           | Start a new cycle             |
| `t`           | Test the notification sound   |
| `m`           | Mute / unmute                 |
| `i`           | Toggle info panel             |
| `o`           | Open options panel            |
| `q`/`esc`     | Quit                          |

In the options panel:

| Key                    | Action                 |
|------------------------|------------------------|
| `↑` / `↓` (`j`/`k`)    | Select field           |
| `←` / `→` (`h`/`l`, `-`/`+`) | Adjust value    |
| `d`                    | Restore defaults       |
| `o` / `esc`            | Close options          |

## Configure it James Brown

Options are stored in a small JSON file at the OS-native config location:

- Linux: `~/.config/pomodoro-tui/config.json`
- macOS: `~/Library/Application Support/pomodoro-tui/config.json`
- Windows: `%AppData%\pomodoro-tui\config.json`

## Dependencies

Direct Go dependencies:

- [`charm.land/bubbletea/v2`][bubbletea] — MIT
- [`charm.land/lipgloss/v2`][lipgloss] — MIT
- [`github.com/gopxl/beep/v2`][beep] — MIT (FLAC decoding + mixer)
- `github.com/gopxl/beep/v2/flac`, `/speaker` — submodules of the above
- [`github.com/ebitengine/oto/v3`][oto] — Apache-2.0 (cross-platform audio output)

Transitive dependencies: see [`go.mod`](go.mod). All are MIT, BSD or
Apache-2.0.

## Hey, who made that awesome sound? 

Notification sound: *"Submarine sonar ping"* by **therealdeevee** on Freesound,
licensed under [Creative Commons 0 (Public Domain)][cc0].
Source: <https://freesound.org/people/therealdeevee/sounds/653896/>

## License

Code in this repository is released under the MIT License; see
[LICENSE](LICENSE). The bundled audio file is CC0 and does not impose any
additional restrictions on the project.

[bubbletea]: https://github.com/charmbracelet/bubbletea
[lipgloss]: https://github.com/charmbracelet/lipgloss
[beep]: https://github.com/gopxl/beep
[oto]: https://github.com/ebitengine/oto
[goreleaser]: https://goreleaser.com
[releases]: https://github.com/Otrivin/pomodoro-tui/releases
[cc0]: https://creativecommons.org/publicdomain/zero/1.0/
