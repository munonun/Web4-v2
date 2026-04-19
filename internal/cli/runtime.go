package cli

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	web4crypto "web4/crypto"
	"web4/node"
	"web4/protocol"
	"web4/transport"
)

const (
	controlServiceName = "Web4Control"
	controlSocketEnv   = "WEB4_CONTROL_SOCK"
	defaultDevAmount   = 1_000_000
	defaultValueExpiry = 1_800_000_000
)

type runtimeConfig struct {
	ID          string
	Listen      string
	Peers       []string
	ControlPath string
}

type managedRuntime struct {
	mu sync.Mutex

	config       runtimeConfig
	node         *node.Node
	transport    *transport.TCPTransport
	control      net.Listener
	rpcServer    *rpc.Server
	acceptDone   chan struct{}
	closed       bool
	devKeys      map[web4crypto.Hash]ed25519.PrivateKey
	seededValue  web4crypto.Value
	seededOwner  string
	controlPath  string
	listenAddr   string
	closeControl sync.Once
}

type controlClient struct {
	rpc  *rpc.Client
	conn io.Closer
}

type ControlService struct {
	runtime *managedRuntime
}

type StatusReply struct {
	ID                   string
	Listen               string
	PeerCount            int
	TransferCount        int
	ConflictSetCount     int
	SelectedLineageCount int
	ControlPath          string
	SeededValueID        string
	SeededOwner          string
}

type SendTxRequest struct {
	InputID string
	Amount  uint64
	Unit    string
	To      string
}

type SendTxReply struct {
	TxID       string
	Status     string
	Confidence float64
}

type ShowTxReply struct {
	TxID       string
	Inputs     []string
	Outputs    []ValueInfo
	Status     string
	Confidence float64
}

type SelectedReply struct {
	Entries []SelectedInfo
}

type SelectedInfo struct {
	InputID string
	TxID    string
}

type FinalityReply struct {
	Entries []FinalityInfo
}

type FinalityInfo struct {
	TxID       string
	Confidence float64
	Status     string
}

type ConflictsReply struct {
	Entries []ConflictInfo
}

type ConflictInfo struct {
	InputID string
	TxIDs   []string
}

type ValuesReply struct {
	Values []ValueInfo
}

type ValueInfo struct {
	ID     string
	Amount uint64
	Unit   string
	Owner  string
	Expiry int64
	Depth  int
}

type StringArg struct {
	Value string
}

func startManagedRuntime(config runtimeConfig) (*managedRuntime, error) {
	if config.ID == "" || config.Listen == "" {
		return nil, fmt.Errorf("node id and listen address are required")
	}
	if config.ControlPath == "" {
		config.ControlPath = controlSocketPath()
	}

	rt := &managedRuntime{
		config:      config,
		node:        node.NewNode(node.PeerID(config.ID)),
		transport:   transport.NewTCPTransport(config.Listen),
		rpcServer:   rpc.NewServer(),
		acceptDone:  make(chan struct{}),
		devKeys:     make(map[web4crypto.Hash]ed25519.PrivateKey),
		controlPath: config.ControlPath,
	}
	rt.node.SetNowFunc(func() int64 { return time.Now().Unix() })
	for _, peer := range config.Peers {
		rt.node.AddPeerID(node.PeerID(peer))
	}
	rt.seedDeveloperValue()

	control, err := listenControlSocket(config.ControlPath)
	if err != nil {
		return nil, err
	}
	rt.control = control

	rt.node.SetSender(func(to node.PeerID, msg protocol.Message) error {
		return rt.transport.Send(string(to), msg)
	})
	if err := rt.transport.Start(func(from string, msg protocol.Message) error {
		rt.mu.Lock()
		defer rt.mu.Unlock()
		return rt.node.OnMessage(node.PeerID(from), msg)
	}); err != nil {
		_ = control.Close()
		_ = os.Remove(config.ControlPath)
		return nil, err
	}
	rt.listenAddr = rt.transport.Addr()

	if err := rt.rpcServer.RegisterName(controlServiceName, &ControlService{runtime: rt}); err != nil {
		_ = rt.transport.Close()
		_ = control.Close()
		_ = os.Remove(config.ControlPath)
		return nil, err
	}

	go rt.acceptLoop()
	return rt, nil
}

func (rt *managedRuntime) acceptLoop() {
	defer close(rt.acceptDone)
	for {
		conn, err := rt.control.Accept()
		if err != nil {
			if rt.isClosed() || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		go rt.rpcServer.ServeConn(conn)
	}
}

func (rt *managedRuntime) Wait(w io.Writer) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	<-signals
	fprintf(w, "shutting down\n")
	return rt.Close()
}

func (rt *managedRuntime) Close() error {
	rt.mu.Lock()
	if rt.closed {
		rt.mu.Unlock()
		return nil
	}
	rt.closed = true
	control := rt.control
	rt.mu.Unlock()

	var firstErr error
	if control != nil {
		if err := control.Close(); err != nil && !errors.Is(err, net.ErrClosed) && firstErr == nil {
			firstErr = err
		}
	}
	if err := rt.transport.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	<-rt.acceptDone
	if err := os.Remove(rt.controlPath); err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (rt *managedRuntime) isClosed() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.closed
}

func (rt *managedRuntime) Status() (StatusReply, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return StatusReply{
		ID:                   rt.config.ID,
		Listen:               rt.listenAddr,
		PeerCount:            rt.node.PeerCount(),
		TransferCount:        rt.node.TransferCount(),
		ConflictSetCount:     rt.node.ConflictSetCount(),
		SelectedLineageCount: rt.node.SelectedLineageCount(),
		ControlPath:          rt.controlPath,
		SeededValueID:        encodeHash(rt.seededValue.ID),
		SeededOwner:          rt.seededOwner,
	}, nil
}

func (rt *managedRuntime) seedDeveloperValue() {
	pub, priv := developerKeys(rt.config.ID)
	value := web4crypto.Value{
		Amount: defaultDevAmount,
		Unit:   "WEB4",
		Owner:  append([]byte(nil), pub...),
		Expiry: defaultValueExpiry,
		Depth:  0,
	}
	value.ID = web4crypto.ComputeValueID(value)
	rt.node.SeedValue(value)
	rt.devKeys[value.ID] = priv
	rt.seededValue = value
	rt.seededOwner = hex.EncodeToString(pub)
}

func (rt *managedRuntime) sendTx(req SendTxRequest) (SendTxReply, error) {
	inputID, err := decodeHash(req.InputID)
	if err != nil {
		return SendTxReply{}, err
	}
	recipient, err := decodeRecipient(req.To)
	if err != nil {
		return SendTxReply{}, err
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	input, ok := rt.node.Value(inputID)
	if !ok {
		return SendTxReply{}, fmt.Errorf("unknown input: %s", req.InputID)
	}
	if input.Amount < req.Amount {
		return SendTxReply{}, fmt.Errorf("amount exceeds input value")
	}
	if input.Unit != req.Unit {
		return SendTxReply{}, fmt.Errorf("input unit mismatch: have %s want %s", input.Unit, req.Unit)
	}
	priv, ok := rt.devKeys[inputID]
	if !ok {
		return SendTxReply{}, fmt.Errorf("no local signing key for input: %s", req.InputID)
	}

	output := web4crypto.Value{
		Amount: req.Amount,
		Unit:   req.Unit,
		Owner:  recipient,
		Expiry: input.Expiry,
		Depth:  input.Depth + 1,
	}
	output.ID = web4crypto.ComputeValueID(output)
	transfer := web4crypto.Transfer{
		Inputs:    []web4crypto.Hash{input.ID},
		Outputs:   []web4crypto.Value{output},
		Timestamp: time.Now().Unix(),
	}
	transfer.Sig = web4crypto.SignCanonicalTransfer(priv, transfer)
	if err := rt.node.AcceptLocalTransfer(transfer); err != nil {
		return SendTxReply{}, err
	}

	txID := web4crypto.ComputeTransferID(transfer)
	state, _ := rt.node.FinalityForTransfer(txID)
	return SendTxReply{TxID: encodeHash(txID), Status: string(state.Status), Confidence: state.Confidence}, nil
}

func (rt *managedRuntime) showTx(txIDHex string) (ShowTxReply, error) {
	txID, err := decodeHash(txIDHex)
	if err != nil {
		return ShowTxReply{}, err
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	tx, ok := rt.node.Transfer(txID)
	if !ok {
		return ShowTxReply{}, fmt.Errorf("unknown transfer: %s", txIDHex)
	}
	state, _ := rt.node.FinalityForTransfer(txID)
	inputs := make([]string, len(tx.Inputs))
	for i, input := range tx.Inputs {
		inputs[i] = encodeHash(input)
	}
	outputs := make([]ValueInfo, len(tx.Outputs))
	for i, output := range tx.Outputs {
		outputs[i] = ValueInfo{
			ID:     encodeHash(output.ID),
			Amount: output.Amount,
			Unit:   output.Unit,
			Owner:  hex.EncodeToString(output.Owner),
			Expiry: output.Expiry,
			Depth:  output.Depth,
		}
	}
	return ShowTxReply{TxID: encodeHash(txID), Inputs: inputs, Outputs: outputs, Status: string(state.Status), Confidence: state.Confidence}, nil
}

func (rt *managedRuntime) selectedState() SelectedReply {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	entries := rt.node.SelectedLineages()
	reply := SelectedReply{Entries: make([]SelectedInfo, len(entries))}
	for i, entry := range entries {
		reply.Entries[i] = SelectedInfo{InputID: encodeHash(entry.Input), TxID: encodeHash(entry.TxID)}
	}
	return reply
}

func (rt *managedRuntime) finalityState() FinalityReply {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	entries := rt.node.FinalityEntries()
	reply := FinalityReply{Entries: make([]FinalityInfo, len(entries))}
	for i, entry := range entries {
		reply.Entries[i] = FinalityInfo{TxID: encodeHash(entry.TxID), Confidence: entry.State.Confidence, Status: string(entry.State.Status)}
	}
	return reply
}

func (rt *managedRuntime) conflictState() ConflictsReply {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	entries := rt.node.ConflictSetEntries()
	reply := ConflictsReply{Entries: make([]ConflictInfo, len(entries))}
	for i, entry := range entries {
		txIDs := make([]string, len(entry.TxIDs))
		for j, txID := range entry.TxIDs {
			txIDs[j] = encodeHash(txID)
		}
		reply.Entries[i] = ConflictInfo{InputID: encodeHash(entry.Input), TxIDs: txIDs}
	}
	return reply
}

func (rt *managedRuntime) valuesState() ValuesReply {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	values := rt.node.Values()
	reply := ValuesReply{Values: make([]ValueInfo, len(values))}
	for i, value := range values {
		reply.Values[i] = ValueInfo{
			ID:     encodeHash(value.ID),
			Amount: value.Amount,
			Unit:   value.Unit,
			Owner:  hex.EncodeToString(value.Owner),
			Expiry: value.Expiry,
			Depth:  value.Depth,
		}
	}
	return reply
}

func (s *ControlService) Status(_ struct{}, reply *StatusReply) error {
	status, err := s.runtime.Status()
	if err != nil {
		return err
	}
	*reply = status
	return nil
}

func (s *ControlService) SendTx(req SendTxRequest, reply *SendTxReply) error {
	resp, err := s.runtime.sendTx(req)
	if err != nil {
		return err
	}
	*reply = resp
	return nil
}

func (s *ControlService) ShowTx(arg StringArg, reply *ShowTxReply) error {
	resp, err := s.runtime.showTx(arg.Value)
	if err != nil {
		return err
	}
	*reply = resp
	return nil
}

func (s *ControlService) Selected(_ struct{}, reply *SelectedReply) error {
	*reply = s.runtime.selectedState()
	return nil
}

func (s *ControlService) Finality(_ struct{}, reply *FinalityReply) error {
	*reply = s.runtime.finalityState()
	return nil
}

func (s *ControlService) Conflicts(_ struct{}, reply *ConflictsReply) error {
	*reply = s.runtime.conflictState()
	return nil
}

func (s *ControlService) Values(_ struct{}, reply *ValuesReply) error {
	*reply = s.runtime.valuesState()
	return nil
}

func dialControlClient(socketPath string) (*controlClient, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("unable to reach local node at %s: %w", socketPath, err)
	}
	return &controlClient{rpc: rpc.NewClient(conn), conn: conn}, nil
}

func (c *controlClient) Close() error {
	var firstErr error
	if err := c.rpc.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := c.conn.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (c *controlClient) Status() (StatusReply, error) {
	var reply StatusReply
	err := c.rpc.Call(controlServiceName+".Status", struct{}{}, &reply)
	return reply, err
}

func (c *controlClient) SendTx(req SendTxRequest) (SendTxReply, error) {
	var reply SendTxReply
	err := c.rpc.Call(controlServiceName+".SendTx", req, &reply)
	return reply, err
}

func (c *controlClient) ShowTx(txID string) (ShowTxReply, error) {
	var reply ShowTxReply
	err := c.rpc.Call(controlServiceName+".ShowTx", StringArg{Value: txID}, &reply)
	return reply, err
}

func (c *controlClient) SelectedState() (SelectedReply, error) {
	var reply SelectedReply
	err := c.rpc.Call(controlServiceName+".Selected", struct{}{}, &reply)
	return reply, err
}

func (c *controlClient) FinalityState() (FinalityReply, error) {
	var reply FinalityReply
	err := c.rpc.Call(controlServiceName+".Finality", struct{}{}, &reply)
	return reply, err
}

func (c *controlClient) ConflictState() (ConflictsReply, error) {
	var reply ConflictsReply
	err := c.rpc.Call(controlServiceName+".Conflicts", struct{}{}, &reply)
	return reply, err
}

func (c *controlClient) ValuesState() (ValuesReply, error) {
	var reply ValuesReply
	err := c.rpc.Call(controlServiceName+".Values", struct{}{}, &reply)
	return reply, err
}

func controlSocketPath() string {
	if path := os.Getenv(controlSocketEnv); path != "" {
		return path
	}
	return defaultControlSocketPath()
}

func startControlSocketPath(explicit, id string) string {
	if explicit != "" {
		return explicit
	}
	if path := os.Getenv(controlSocketEnv); path != "" {
		return path
	}
	return derivedControlSocketPath(id)
}

func clientControlSocketPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return controlSocketPath()
}

func defaultControlSocketPath() string {
	return filepath.Join(os.TempDir(), "web4-cli.sock")
}

func derivedControlSocketPath(id string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("web4-%s.sock", sanitizeControlSocketID(id)))
}

func sanitizeControlSocketID(id string) string {
	if id == "" {
		return "cli"
	}
	var out []rune
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r)
		case r >= '0' && r <= '9':
			out = append(out, r)
		case r == '-', r == '_', r == '.':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}

func listenControlSocket(socketPath string) (net.Listener, error) {
	if conn, err := net.Dial("unix", socketPath); err == nil {
		_ = conn.Close()
		return nil, fmt.Errorf("another local node is already running at %s", socketPath)
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return nil, err
	}
	info, err := os.Lstat(socketPath)
	if err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("control path exists and is not a socket: %s", socketPath)
		}
		if err := os.Remove(socketPath); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return net.Listen("unix", socketPath)
}

func developerKeys(id string) (ed25519.PublicKey, ed25519.PrivateKey) {
	seed := sha256.Sum256([]byte("web4-dev:" + id))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pub, priv[32:])
	return pub, priv
}

func decodeHash(value string) (web4crypto.Hash, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return web4crypto.Hash{}, err
	}
	if len(decoded) != len(web4crypto.Hash{}) {
		return web4crypto.Hash{}, fmt.Errorf("invalid hash length: %d", len(decoded))
	}
	var hash web4crypto.Hash
	copy(hash[:], decoded)
	return hash, nil
}

func decodeRecipient(value string) ([]byte, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid recipient public key length: %d", len(decoded))
	}
	return decoded, nil
}

func encodeHash(hash web4crypto.Hash) string {
	return hex.EncodeToString(hash[:])
}

func sortStrings(values []string) []string {
	cloned := append([]string(nil), values...)
	sort.Strings(cloned)
	return cloned
}
