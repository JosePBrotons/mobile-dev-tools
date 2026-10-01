// Package shellenv holds the shell environment every step runs in and
// idempotent edits to the user's shell profile.
package shellenv

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prelude is prepended to every check and install script. Each step runs
// in a fresh bash, so it loads tools that earlier steps may have installed.
const Prelude = `if [ -x /opt/homebrew/bin/brew ]; then eval "$(/opt/homebrew/bin/brew shellenv)"; ` +
	`elif [ -x /usr/local/bin/brew ]; then eval "$(/usr/local/bin/brew shellenv)"; fi
export NVM_DIR="$HOME/.nvm"
if [ -s "$NVM_DIR/nvm.sh" ]; then . "$NVM_DIR/nvm.sh"; fi
`

// ProfileFile is the shell profile mdt edits, relative to the home folder.
const ProfileFile = ".zprofile"

// AndroidHomeLines match Misc/set-android-home-path.sh. They are written
// literally so they expand when the profile loads.
var AndroidHomeLines = []string{
	`export ANDROID_HOME=$HOME/Library/Android/sdk`,
	`export PATH=$PATH:$ANDROID_HOME/emulator`,
	`export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin`,
	`export PATH=$PATH:$ANDROID_HOME/platform-tools`,
}

// EnsureLines appends lines to path unless its first line is already there.
// The file is created if missing. It reports whether the file changed.
func EnsureLines(path string, lines []string) (bool, error) {
	if len(lines) == 0 {
		return false, nil
	}
	present, err := hasLine(path, lines[0])
	if err != nil || present {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	content := strings.Join(lines, "\n") + "\n"
	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		content = "\n" + content
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, f.Close()
}

func hasLine(path, line string) (bool, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if sc.Text() == line {
			return true, nil
		}
	}
	return false, sc.Err()
}
