package cli

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var ansiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestRunNoArgsEntersShellMode(t *testing.T) {
	output := cleanShellOutput(runShellInput(t, "exit\n"))
	if !strings.Contains(output, "type \"help\" for commands") {
		t.Fatalf("missing shell hint\n%s", output)
	}
	if !strings.Contains(output, "> ") {
		t.Fatalf("missing prompt\n%s", output)
	}
	if !strings.Contains(output, "WEB4") && !strings.Contains(output, "____") {
		t.Fatalf("missing banner\n%s", output)
	}
}

func TestShellHelpCommand(t *testing.T) {
	output := cleanShellOutput(runShellInput(t, "help\nexit\n"))
	for _, fragment := range []string{
		"Commands:",
		"node start",
		"node status",
		"tx send",
		"tx show",
		"state selected",
		"state finality",
		"state conflicts",
		"state values",
		"demo tcp-basic",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("missing help fragment %q\n%s", fragment, output)
		}
	}
}

func TestShellExitCommandExitsCleanly(t *testing.T) {
	var buf bytes.Buffer
	if err := runWithIO(nil, strings.NewReader("quit\n"), &buf); err != nil {
		t.Fatalf("runWithIO: %v", err)
	}
	output := cleanShellOutput(buf.String())
	if strings.Contains(output, "error:") {
		t.Fatalf("unexpected shell error output\n%s", output)
	}
	if !strings.Contains(output, "> ") {
		t.Fatalf("expected prompt in shell output\n%s", output)
	}
}

func TestOneShotCommandsStillWork(t *testing.T) {
	var buf bytes.Buffer
	if err := Run([]string{"demo", "basic"}, &buf); err != nil {
		t.Fatalf("Run(demo basic): %v", err)
	}
	if !strings.Contains(buf.String(), "[A] created tx:") {
		t.Fatalf("unexpected one-shot output\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "type \"help\" for commands") {
		t.Fatalf("one-shot command should not enter shell mode\n%s", buf.String())
	}
}

func runShellInput(t *testing.T, input string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := runWithIO(nil, strings.NewReader(input), &buf); err != nil {
		t.Fatalf("runWithIO: %v", err)
	}
	return buf.String()
}

func cleanShellOutput(output string) string {
	output = ansiSequence.ReplaceAllString(output, "")
	output = strings.ReplaceAll(output, "\r", "")
	output = strings.ReplaceAll(output, "\b", "")
	return output
}
