package cli

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIStatusSendAndStateCommands(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "web4-cli.sock")
	rt, err := startManagedRuntime(runtimeConfig{ID: "dev-a", Listen: "127.0.0.1:0", ControlPath: socketPath})
	if err != nil {
		t.Fatalf("startManagedRuntime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close() })

	previous := os.Getenv(controlSocketEnv)
	if err := os.Setenv(controlSocketEnv, socketPath); err != nil {
		t.Fatalf("Setenv: %v", err)
	}
	t.Cleanup(func() {
		if previous == "" {
			_ = os.Unsetenv(controlSocketEnv)
			return
		}
		_ = os.Setenv(controlSocketEnv, previous)
	})

	statusBefore := runCLI(t, "node", "status")
	if !strings.Contains(statusBefore, "node_id: dev-a") {
		t.Fatalf("missing node id in status\n%s", statusBefore)
	}
	if !strings.Contains(statusBefore, "tx_store_count: 0") {
		t.Fatalf("unexpected initial transfer count\n%s", statusBefore)
	}

	status, err := rt.Status()
	if err != nil {
		t.Fatalf("runtime status: %v", err)
	}
	recipient, _ := developerKeys("recipient")
	toHex := hex.EncodeToString(recipient)

	sendOutput := runCLI(t,
		"tx", "send",
		"--input", status.SeededValueID,
		"--amount", "25",
		"--unit", "WEB4",
		"--to", toHex,
	)
	if !strings.Contains(sendOutput, "tx_id: ") || !strings.Contains(sendOutput, "status: SELECTED") {
		t.Fatalf("unexpected tx send output\n%s", sendOutput)
	}
	txID := outputField(t, sendOutput, "tx_id: ")

	showOutput := runCLI(t, "tx", "show", txID)
	if !strings.Contains(showOutput, "tx_id: "+txID) {
		t.Fatalf("missing tx id in tx show\n%s", showOutput)
	}
	if !strings.Contains(showOutput, "status: SELECTED") {
		t.Fatalf("missing finality status in tx show\n%s", showOutput)
	}

	selectedOutput := runCLI(t, "state", "selected")
	if !strings.Contains(selectedOutput, txID) {
		t.Fatalf("selected state missing tx id\n%s", selectedOutput)
	}

	finalityOutput := runCLI(t, "state", "finality")
	if !strings.Contains(finalityOutput, txID) || !strings.Contains(finalityOutput, "status=SELECTED") {
		t.Fatalf("finality state missing tx\n%s", finalityOutput)
	}

	conflictsOutput := runCLI(t, "state", "conflicts")
	if !strings.Contains(conflictsOutput, txID) {
		t.Fatalf("conflicts state missing tx\n%s", conflictsOutput)
	}

	valuesOutput := runCLI(t, "state", "values")
	if !strings.Contains(valuesOutput, status.SeededValueID) {
		t.Fatalf("values state missing seeded input\n%s", valuesOutput)
	}
	if strings.Count(valuesOutput, "amount=") < 2 {
		t.Fatalf("values state should include input and output\n%s", valuesOutput)
	}

	statusAfter := runCLI(t, "node", "status")
	if !strings.Contains(statusAfter, "tx_store_count: 1") {
		t.Fatalf("unexpected transfer count after send\n%s", statusAfter)
	}
	if !strings.Contains(statusAfter, "selected_lineage_count: 1") {
		t.Fatalf("unexpected selected lineage count after send\n%s", statusAfter)
	}
}

func TestMultipleRuntimesCanRunWithDifferentControlSockets(t *testing.T) {
	t.Setenv(controlSocketEnv, "")
	socketA := filepath.Join(t.TempDir(), "web4-A.sock")
	socketB := filepath.Join(t.TempDir(), "web4-B.sock")
	rtA := startTestRuntime(t, "node-A", socketA)
	rtB := startTestRuntime(t, "node-B", socketB)

	statusA, err := rtA.Status()
	if err != nil {
		t.Fatalf("rtA.Status: %v", err)
	}
	statusB, err := rtB.Status()
	if err != nil {
		t.Fatalf("rtB.Status: %v", err)
	}
	if statusA.ControlPath != socketA {
		t.Fatalf("unexpected control path for A: got %s want %s", statusA.ControlPath, socketA)
	}
	if statusB.ControlPath != socketB {
		t.Fatalf("unexpected control path for B: got %s want %s", statusB.ControlPath, socketB)
	}

	statusOutputA := runCLI(t, "node", "status", "--control", socketA)
	statusOutputB := runCLI(t, "node", "status", "--control", socketB)
	if !strings.Contains(statusOutputA, "node_id: node-A") {
		t.Fatalf("status A missing node id\n%s", statusOutputA)
	}
	if !strings.Contains(statusOutputB, "node_id: node-B") {
		t.Fatalf("status B missing node id\n%s", statusOutputB)
	}
}

func TestClientCommandsCanTargetDifferentSockets(t *testing.T) {
	t.Setenv(controlSocketEnv, "")
	socketA := filepath.Join(t.TempDir(), "web4-A.sock")
	socketB := filepath.Join(t.TempDir(), "web4-B.sock")
	rtA := startTestRuntime(t, "node-A", socketA)
	rtB := startTestRuntime(t, "node-B", socketB)

	statusA, err := rtA.Status()
	if err != nil {
		t.Fatalf("rtA.Status: %v", err)
	}
	statusB, err := rtB.Status()
	if err != nil {
		t.Fatalf("rtB.Status: %v", err)
	}
	recipient, _ := developerKeys("recipient")
	toHex := hex.EncodeToString(recipient)

	outputA := runCLI(t,
		"tx", "send",
		"--control", socketA,
		"--input", statusA.SeededValueID,
		"--amount", "25",
		"--unit", "WEB4",
		"--to", toHex,
	)
	txIDA := outputField(t, outputA, "tx_id: ")

	outputB := runCLI(t,
		"tx", "send",
		"--control", socketB,
		"--input", statusB.SeededValueID,
		"--amount", "40",
		"--unit", "WEB4",
		"--to", toHex,
	)
	txIDB := outputField(t, outputB, "tx_id: ")

	if txIDA == txIDB {
		t.Fatal("different runtimes should not return identical tx ids in this test")
	}

	showA := runCLI(t, "tx", "show", "--control", socketA, txIDA)
	if !strings.Contains(showA, "tx_id: "+txIDA) {
		t.Fatalf("tx show A missing tx id\n%s", showA)
	}
	showB := runCLI(t, "tx", "show", "--control", socketB, txIDB)
	if !strings.Contains(showB, "tx_id: "+txIDB) {
		t.Fatalf("tx show B missing tx id\n%s", showB)
	}

	finalityA := runCLI(t, "state", "finality", "--control", socketA)
	if !strings.Contains(finalityA, txIDA) || strings.Contains(finalityA, txIDB) {
		t.Fatalf("unexpected finality output for A\n%s", finalityA)
	}
	finalityB := runCLI(t, "state", "finality", "--control", socketB)
	if !strings.Contains(finalityB, txIDB) || strings.Contains(finalityB, txIDA) {
		t.Fatalf("unexpected finality output for B\n%s", finalityB)
	}
}

func TestDefaultControlPathDerivedFromNodeID(t *testing.T) {
	t.Setenv(controlSocketEnv, "")
	got := startControlSocketPath("", "node-A.test")
	want := filepath.Join(os.TempDir(), "web4-node-A.test.sock")
	if got != want {
		t.Fatalf("unexpected derived control path: got %s want %s", got, want)
	}
}

func TestExplicitControlOverridesEnvironmentFallback(t *testing.T) {
	socketA := filepath.Join(t.TempDir(), "web4-A.sock")
	socketB := filepath.Join(t.TempDir(), "web4-B.sock")
	startTestRuntime(t, "node-A", socketA)
	startTestRuntime(t, "node-B", socketB)
	t.Setenv(controlSocketEnv, socketA)

	statusOutput := runCLI(t, "node", "status", "--control", socketB)
	if !strings.Contains(statusOutput, "node_id: node-B") {
		t.Fatalf("explicit control path did not override env fallback\n%s", statusOutput)
	}
	if got := startControlSocketPath(socketB, "node-A"); got != socketB {
		t.Fatalf("explicit start control override failed: got %s want %s", got, socketB)
	}
}

func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Run(args, &buf); err != nil {
		t.Fatalf("Run(%v): %v", args, err)
	}
	return buf.String()
}

func outputField(t *testing.T, output, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	t.Fatalf("missing field %q in output\n%s", prefix, output)
	return ""
}

func startTestRuntime(t *testing.T, id, socketPath string) *managedRuntime {
	t.Helper()
	rt, err := startManagedRuntime(runtimeConfig{ID: id, Listen: "127.0.0.1:0", ControlPath: socketPath})
	if err != nil {
		t.Fatalf("startManagedRuntime(%s): %v", id, err)
	}
	t.Cleanup(func() { _ = rt.Close() })
	return rt
}
