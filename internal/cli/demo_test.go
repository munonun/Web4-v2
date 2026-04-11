package cli

import (
	"bytes"
	"io"
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

func runDemo(t *testing.T, fn func(w io.Writer) error) string {
	t.Helper()
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		t.Fatalf("demo failed: %v", err)
	}
	return buf.String()
}
