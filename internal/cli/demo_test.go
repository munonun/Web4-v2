package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRunDemoBasicDeterministic(t *testing.T) {
	first := runDemo(t, RunDemoBasic)
	second := runDemo(t, RunDemoBasic)
	if first != second {
		t.Fatal("basic demo output must be deterministic")
	}
}

func TestRunDemoConflictDeterministic(t *testing.T) {
	first := runDemo(t, RunDemoConflict)
	second := runDemo(t, RunDemoConflict)
	if first != second {
		t.Fatal("conflict demo output must be deterministic")
	}
}

func TestRunDemoProofDeterministic(t *testing.T) {
	first := runDemo(t, RunDemoProof)
	second := runDemo(t, RunDemoProof)
	if first != second {
		t.Fatal("proof demo output must be deterministic")
	}
}

func TestRunDemoTCPBasic(t *testing.T) {
	output := runTCPDemo(t, []string{"127.0.0.1:0", "127.0.0.1:0", "127.0.0.1:0"})

	required := []string{
		"[TCP BASIC]\n",
		"[A] listening on 127.0.0.1:",
		"[B] listening on 127.0.0.1:",
		"[C] listening on 127.0.0.1:",
		"[A] created tx: ",
		"[A -> B] INV ",
		"[B -> A] GET ",
		"[A -> B] TX ",
		"[B] accepted tx ",
		"[A -> C] INV ",
		"[C -> A] GET ",
		"[A -> C] TX ",
		"[C] accepted tx ",
		"Final selected state:\n",
		"  Node A -> ",
		"  Node B -> ",
		"  Node C -> ",
	}
	for _, fragment := range required {
		if !strings.Contains(output, fragment) {
			t.Fatalf("tcp-basic output missing %q\n%s", fragment, output)
		}
	}

	ordered := []string{
		"[TCP BASIC]",
		"[A] listening on 127.0.0.1:",
		"[B] listening on 127.0.0.1:",
		"[C] listening on 127.0.0.1:",
		"[A] created tx: ",
		"[A -> B] INV ",
		"[B -> A] GET ",
		"[A -> B] TX ",
		"[B] accepted tx ",
		"[A -> C] INV ",
		"[C -> A] GET ",
		"[A -> C] TX ",
		"[C] accepted tx ",
		"Final selected state:",
		"  Node A -> ",
		"  Node B -> ",
		"  Node C -> ",
	}
	last := -1
	for _, fragment := range ordered {
		idx := strings.Index(output, fragment)
		if idx == -1 {
			t.Fatalf("tcp-basic output missing ordered fragment %q\n%s", fragment, output)
		}
		if idx < last {
			t.Fatalf("tcp-basic output out of order around %q\n%s", fragment, output)
		}
		last = idx
	}

	if strings.Count(output, "(SELECTED)") != 3 {
		t.Fatalf("expected three SELECTED finality states\n%s", output)
	}
	if strings.Count(output, "accepted tx") != 2 {
		t.Fatalf("expected two accepted tx lines\n%s", output)
	}
}

func runDemo(t *testing.T, fn func(w io.Writer) error) string {
	t.Helper()
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		t.Fatalf("demo failed: %v", err)
	}
	return buf.String()
}

func runTCPDemo(t *testing.T, addrs []string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := runDemoTCPBasic(&buf, addrs); err != nil {
		t.Fatalf("tcp demo failed: %v", err)
	}
	return buf.String()
}
