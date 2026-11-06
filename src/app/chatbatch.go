package main

import (
	"fmt"
	"strings"
)

// BatchResultDTO is the outcome of running several commands one after another.
type BatchResultDTO struct {
	Total   int      `json:"total"`   // commands in the script (blank and comment lines not counted)
	Done    int      `json:"done"`    // commands executed successfully
	Failed  int      `json:"failed"`  // script line (1-based) of the command that failed, 0 if none
	Stopped int      `json:"stopped"` // script line (1-based) of a quit/exit/stop command, 0 if none
	Error   string   `json:"error"`   // the error of the failed command
	Log     []string `json:"log"`     // one entry per executed command: "line 3: <command> -> <answer>"
}

// ChatBatch runs a script of commands, one per line, in order (batch mode of
// the command bar). Blank lines and lines starting with # or // are skipped. The
// batch STOPS at the first command that fails, and at a line that says quit,
// exit or stop; the commands before it stay applied. The whole batch is ONE undo
// step: a single Cmd-Z takes back everything it did.
func (a *App) ChatBatch(script string) BatchResultDTO {
	type cmd struct {
		line int
		text string
	}
	var cmds []cmd
	for i, raw := range strings.Split(strings.ReplaceAll(script, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//") {
			continue
		}
		cmds = append(cmds, cmd{i + 1, t})
	}
	res := BatchResultDTO{Total: len(cmds), Log: []string{}}
	if len(cmds) == 0 {
		return res
	}
	mark := len(a.undoStack)
	a.pushUndo() // the state before the batch
	for _, c := range cmds {
		switch strings.ToLower(c.text) {
		case "quit", "exit", "stop":
			res.Stopped = c.line
			res.Log = append(res.Log, fmt.Sprintf("line %d: %s -> batch stopped", c.line, c.text))
			return a.finishBatch(res, mark)
		}
		out, err := a.Chat(c.text)
		if err != nil {
			res.Failed, res.Error = c.line, err.Error()
			res.Log = append(res.Log, fmt.Sprintf("line %d: %s -> ERROR: %v", c.line, c.text, err))
			return a.finishBatch(res, mark)
		}
		res.Done++
		res.Log = append(res.Log, fmt.Sprintf("line %d: %s -> %s", c.line, c.text, firstLine(out)))
	}
	return a.finishBatch(res, mark)
}

// finishBatch collapses the undo steps the batch created into one (the state
// before the batch), or drops it when nothing ran.
func (a *App) finishBatch(res BatchResultDTO, mark int) BatchResultDTO {
	if res.Done == 0 {
		if len(a.undoStack) > mark {
			a.undoStack = a.undoStack[:mark]
		}
		return res
	}
	if len(a.undoStack) > mark+1 {
		a.undoStack = a.undoStack[:mark+1]
	}
	return res
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " ..."
	}
	return s
}
