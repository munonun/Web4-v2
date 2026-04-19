package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func Run(args []string, w io.Writer) error {
	return runWithIO(args, os.Stdin, w)
}

func runWithIO(args []string, in io.Reader, w io.Writer) error {
	if len(args) == 0 {
		return runShell(in, w)
	}
	return executeCommand(args, w, nil)
}

func executeCommand(args []string, w io.Writer, shell *shellSession) error {
	switch args[0] {
	case "demo":
		return runDemoCommand(args[1:], w)
	case "node":
		return runNodeCommand(args[1:], w, shell)
	case "tx":
		return runTxCommand(args[1:], w)
	case "state":
		return runStateCommand(args[1:], w)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runNodeCommand(args []string, w io.Writer, shell *shellSession) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: web4 node [start|status]")
	}

	switch args[0] {
	case "start":
		return runNodeStart(args[1:], w, shell)
	case "status":
		return runNodeStatus(args[1:], w)
	default:
		return fmt.Errorf("unknown node command: %s", args[0])
	}
}

func runTxCommand(args []string, w io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: web4 tx [send|show]")
	}

	switch args[0] {
	case "send":
		return runTxSend(args[1:], w)
	case "show":
		return runTxShow(args[1:], w)
	default:
		return fmt.Errorf("unknown tx command: %s", args[0])
	}
}

func runStateCommand(args []string, w io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: web4 state [selected|finality|conflicts|values]")
	}

	switch args[0] {
	case "selected":
		return runStateSelected(args[1:], w)
	case "finality":
		return runStateFinality(args[1:], w)
	case "conflicts":
		return runStateConflicts(args[1:], w)
	case "values":
		return runStateValues(args[1:], w)
	default:
		return fmt.Errorf("unknown state command: %s", args[0])
	}
}

func runNodeStart(args []string, w io.Writer, shell *shellSession) error {
	fs := flag.NewFlagSet("node start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var id string
	var listen string
	var control string
	var peers stringListFlag
	fs.StringVar(&id, "id", "", "node id")
	fs.StringVar(&listen, "listen", "", "listen address")
	fs.StringVar(&control, "control", "", "control socket path")
	fs.Var(&peers, "peer", "peer address")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("usage: web4 node start --id <id> --listen <addr> [--peer <addr>] [--control <path>]")
	}
	if id == "" || listen == "" {
		return fmt.Errorf("usage: web4 node start --id <id> --listen <addr> [--peer <addr>] [--control <path>]")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: web4 node start --id <id> --listen <addr> [--peer <addr>] [--control <path>]")
	}
	if shell != nil && shell.runtime != nil {
		return fmt.Errorf("node already started in this shell")
	}

	rt, err := startManagedRuntime(runtimeConfig{ID: id, Listen: listen, Peers: peers, ControlPath: startControlSocketPath(control, id)})
	if err != nil {
		return err
	}

	status, err := rt.Status()
	if err != nil {
		_ = rt.Close()
		return err
	}
	fprintf(w, "node started\n")
	fprintf(w, "id: %s\n", status.ID)
	fprintf(w, "listen: %s\n", status.Listen)
	fprintf(w, "peers: %d\n", status.PeerCount)
	fprintf(w, "control: %s\n", status.ControlPath)
	fprintf(w, "seeded_value: %s\n", status.SeededValueID)
	fprintf(w, "seeded_owner: %s\n", status.SeededOwner)
	if shell != nil {
		shell.runtime = rt
		return nil
	}
	defer func() {
		_ = rt.Close()
	}()
	return rt.Wait(w)
}

func runNodeStatus(args []string, w io.Writer) error {
	control, remaining, err := parseControlOnlyArgs("node status", "web4 node status [--control <path>]", args)
	if err != nil {
		return err
	}
	if len(remaining) != 0 {
		return fmt.Errorf("usage: web4 node status [--control <path>]")
	}

	client, err := dialControlClient(clientControlSocketPath(control))
	if err != nil {
		return err
	}
	defer client.Close()

	status, err := client.Status()
	if err != nil {
		return err
	}
	fprintf(w, "node_id: %s\n", status.ID)
	fprintf(w, "listen: %s\n", status.Listen)
	fprintf(w, "peer_count: %d\n", status.PeerCount)
	fprintf(w, "tx_store_count: %d\n", status.TransferCount)
	fprintf(w, "conflict_set_count: %d\n", status.ConflictSetCount)
	fprintf(w, "selected_lineage_count: %d\n", status.SelectedLineageCount)
	return nil
}

func runTxSend(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("tx send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var input string
	var amount uint64
	var unit string
	var to string
	var control string
	fs.StringVar(&control, "control", "", "control socket path")
	fs.StringVar(&input, "input", "", "input value id")
	fs.Uint64Var(&amount, "amount", 0, "amount")
	fs.StringVar(&unit, "unit", "", "unit")
	fs.StringVar(&to, "to", "", "recipient public key hex")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("usage: web4 tx send [--control <path>] --input <value_id> --amount <n> --unit <unit> --to <pubkey_hex>")
	}
	if input == "" || amount == 0 || unit == "" || to == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: web4 tx send [--control <path>] --input <value_id> --amount <n> --unit <unit> --to <pubkey_hex>")
	}

	client, err := dialControlClient(clientControlSocketPath(control))
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.SendTx(SendTxRequest{InputID: input, Amount: amount, Unit: unit, To: to})
	if err != nil {
		return err
	}
	fprintf(w, "tx_id: %s\n", reply.TxID)
	fprintf(w, "status: %s\n", reply.Status)
	fprintf(w, "confidence: %.2f\n", reply.Confidence)
	return nil
}

func runTxShow(args []string, w io.Writer) error {
	control, remaining, err := parseControlOnlyArgs("tx show", "web4 tx show [--control <path>] <tx_id>", args)
	if err != nil {
		return err
	}
	if len(remaining) != 1 {
		return fmt.Errorf("usage: web4 tx show [--control <path>] <tx_id>")
	}

	client, err := dialControlClient(clientControlSocketPath(control))
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.ShowTx(remaining[0])
	if err != nil {
		return err
	}
	fprintf(w, "tx_id: %s\n", reply.TxID)
	fprintf(w, "inputs:\n")
	for _, input := range reply.Inputs {
		fprintf(w, "  %s\n", input)
	}
	fprintf(w, "outputs:\n")
	for _, output := range reply.Outputs {
		fprintf(w, "  value_id=%s amount=%d unit=%s owner=%s expiry=%d depth=%d\n",
			output.ID,
			output.Amount,
			output.Unit,
			output.Owner,
			output.Expiry,
			output.Depth,
		)
	}
	fprintf(w, "status: %s\n", reply.Status)
	fprintf(w, "confidence: %.2f\n", reply.Confidence)
	return nil
}

func runStateSelected(args []string, w io.Writer) error {
	control, remaining, err := parseControlOnlyArgs("state selected", "web4 state selected [--control <path>]", args)
	if err != nil {
		return err
	}
	if len(remaining) != 0 {
		return fmt.Errorf("usage: web4 state selected [--control <path>]")
	}

	client, err := dialControlClient(clientControlSocketPath(control))
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.SelectedState()
	if err != nil {
		return err
	}
	for _, entry := range reply.Entries {
		fprintf(w, "%s -> %s\n", entry.InputID, entry.TxID)
	}
	return nil
}

func runStateFinality(args []string, w io.Writer) error {
	control, remaining, err := parseControlOnlyArgs("state finality", "web4 state finality [--control <path>]", args)
	if err != nil {
		return err
	}
	if len(remaining) != 0 {
		return fmt.Errorf("usage: web4 state finality [--control <path>]")
	}

	client, err := dialControlClient(clientControlSocketPath(control))
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.FinalityState()
	if err != nil {
		return err
	}
	for _, entry := range reply.Entries {
		fprintf(w, "%s confidence=%.2f status=%s\n", entry.TxID, entry.Confidence, entry.Status)
	}
	return nil
}

func runStateConflicts(args []string, w io.Writer) error {
	control, remaining, err := parseControlOnlyArgs("state conflicts", "web4 state conflicts [--control <path>]", args)
	if err != nil {
		return err
	}
	if len(remaining) != 0 {
		return fmt.Errorf("usage: web4 state conflicts [--control <path>]")
	}

	client, err := dialControlClient(clientControlSocketPath(control))
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.ConflictState()
	if err != nil {
		return err
	}
	for _, entry := range reply.Entries {
		fprintf(w, "%s: %s\n", entry.InputID, strings.Join(entry.TxIDs, " "))
	}
	return nil
}

func runStateValues(args []string, w io.Writer) error {
	control, remaining, err := parseControlOnlyArgs("state values", "web4 state values [--control <path>]", args)
	if err != nil {
		return err
	}
	if len(remaining) != 0 {
		return fmt.Errorf("usage: web4 state values [--control <path>]")
	}

	client, err := dialControlClient(clientControlSocketPath(control))
	if err != nil {
		return err
	}
	defer client.Close()

	reply, err := client.ValuesState()
	if err != nil {
		return err
	}
	for _, value := range reply.Values {
		fprintf(w, "%s amount=%d unit=%s owner=%s expiry=%d depth=%d\n",
			value.ID,
			value.Amount,
			value.Unit,
			value.Owner,
			value.Expiry,
			value.Depth,
		)
	}
	return nil
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func parseControlOnlyArgs(name, usage string, args []string) (string, []string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var control string
	fs.StringVar(&control, "control", "", "control socket path")
	if err := fs.Parse(args); err != nil {
		return "", nil, fmt.Errorf("usage: %s", usage)
	}
	return control, fs.Args(), nil
}
