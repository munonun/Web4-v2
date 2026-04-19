package cli

import (
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

const web4Banner = `__        ________ ____  _  _
\ \      / / ____// __ \| || |
 \ \ /\ / / __/  / /_/ /| || |_
  \ V  V / /___ / _, _/ |__   _|
   \_/\_/_____/ /_/ |_|    |_|
`

const shellHelpText = `Commands:
  node start --id <id> --listen <addr> [--peer <addr>]
  node status
  tx send --input <value_id> --amount <n> --unit <unit> --to <pubkey_hex>
  tx show <tx_id>
  state selected
  state finality
  state conflicts
  state values
  demo basic
  demo conflict
  demo proof
  demo tcp-basic
  help
  exit
`

type shellSession struct {
	runtime *managedRuntime
}

func runShell(in io.Reader, w io.Writer) error {
	stdin := io.NopCloser(in)
	cfg := &readline.Config{
		Prompt:          "> ",
		Stdin:           stdin,
		Stdout:          w,
		Stderr:          w,
		InterruptPrompt: "^C",
		EOFPrompt:       "",
	}
	if !isInteractiveShellIO(in, w) {
		cfg.ForceUseInteractive = true
		cfg.FuncIsTerminal = func() bool { return true }
		cfg.FuncMakeRaw = func() error { return nil }
		cfg.FuncExitRaw = func() error { return nil }
		cfg.FuncOnWidthChanged = func(func()) {}
		cfg.FuncGetWidth = func() int { return 80 }
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		return err
	}
	defer rl.Close()

	out := rl.Stdout()
	session := &shellSession{}
	defer func() {
		if session.runtime != nil {
			_ = session.runtime.Close()
		}
	}()

	fprintf(out, "%s", web4Banner)
	fprintf(out, "type \"help\" for commands\n")

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if strings.TrimSpace(line) == "" {
				continue
			}
			continue
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		args := strings.Fields(line)
		switch args[0] {
		case "help":
			fprintf(out, "%s", shellHelpText)
			continue
		case "exit", "quit":
			return nil
		case "clear":
			clearShellScreen(rl)
			continue
		}
		if err := executeCommand(args, out, session); err != nil {
			fprintf(out, "error: %v\n", err)
		}
	}
}

func clearShellScreen(rl *readline.Instance) {
	rl.Clean()
	_, _ = readline.ClearScreen(rl.Stdout())
	rl.Refresh()
}

func isInteractiveShellIO(in io.Reader, w io.Writer) bool {
	inFile, inOK := in.(*os.File)
	outFile, outOK := w.(*os.File)
	if !inOK || !outOK {
		return false
	}
	return inFile == os.Stdin && outFile == os.Stdout
}
