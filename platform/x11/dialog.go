//go:build x11 && !nogui

package x11

import (
	"os/exec"
	"strings"
	"sync"

	"github.com/timzifer/lux/platform"
)

// dialogTool caches which dialog tool (zenity or kdialog) is available.
var (
	dialogToolOnce sync.Once
	dialogToolPath string
	dialogToolName string // "zenity" or "kdialog"
)

func findDialogTool() {
	dialogToolOnce.Do(func() {
		if path, err := exec.LookPath("zenity"); err == nil {
			dialogToolPath = path
			dialogToolName = "zenity"
			return
		}
		if path, err := exec.LookPath("kdialog"); err == nil {
			dialogToolPath = path
			dialogToolName = "kdialog"
		}
	})
}

// ShowMessageDialog displays a message dialog via zenity or kdialog.
func (p *Platform) ShowMessageDialog(title, message string, kind platform.DialogKind) error {
	findDialogTool()
	if dialogToolPath == "" {
		return errNoDialogTool
	}

	var args []string
	switch dialogToolName {
	case "zenity":
		flag := "--info"
		switch kind {
		case platform.DialogWarning:
			flag = "--warning"
		case platform.DialogError:
			flag = "--error"
		}
		args = []string{flag, "--title", title, "--text", message}
	case "kdialog":
		method := "--msgbox"
		switch kind {
		case platform.DialogWarning:
			method = "--sorry"
		case platform.DialogError:
			method = "--error"
		}
		args = []string{method, message, "--title", title}
	}

	return exec.Command(dialogToolPath, args...).Run()
}

// ShowConfirmDialog displays a Yes/No dialog via zenity or kdialog.
func (p *Platform) ShowConfirmDialog(title, message string) (bool, error) {
	findDialogTool()
	if dialogToolPath == "" {
		return false, errNoDialogTool
	}

	var args []string
	switch dialogToolName {
	case "zenity":
		args = []string{"--question", "--title", title, "--text", message}
	case "kdialog":
		args = []string{"--yesno", message, "--title", title}
	}

	err := exec.Command(dialogToolPath, args...).Run()
	if err == nil {
		return true, nil
	}
	// Exit code 1 = user clicked No/Cancel, not an error.
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// ShowInputDialog displays a text entry dialog via zenity or kdialog.
func (p *Platform) ShowInputDialog(title, message, defaultValue string) (string, bool, error) {
	findDialogTool()
	if dialogToolPath == "" {
		return "", false, errNoDialogTool
	}

	var args []string
	switch dialogToolName {
	case "zenity":
		args = []string{"--entry", "--title", title, "--text", message, "--entry-text", defaultValue}
	case "kdialog":
		args = []string{"--inputbox", message, defaultValue, "--title", title}
	}

	out, err := exec.Command(dialogToolPath, args...).Output()
	if err == nil {
		return strings.TrimRight(string(out), "\n"), true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return "", false, nil
	}
	return "", false, err
}

var errNoDialogTool = &dialogError{"no dialog tool found (install zenity or kdialog)"}

type dialogError struct{ msg string }

func (e *dialogError) Error() string { return e.msg }
