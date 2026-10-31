package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"weftlink/internal/proto"
)

// Wire types mirror the daemon control responses. The protocol spec is the
// source of truth; these are deliberately thin.
type peer struct {
	PeerID      string `json:"peer_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Version     string `json:"version"`
	Address     string `json:"address"`
	Fingerprint string `json:"fingerprint"`
	LastSeen    int64  `json:"last_seen"`
	Paired      bool   `json:"paired"`
}

type statusResponse struct {
	Type            string `json:"type"`
	NodeID          string `json:"node_id"`
	DisplayName     string `json:"display_name"`
	Role            string `json:"role"`
	Version         string `json:"version"`
	Fingerprint     string `json:"fingerprint"`
	Peers           []peer `json:"peers"`
	PeerCount       int    `json:"peer_count"`
	PairedCount     int    `json:"paired_count"`
	DiscoveryPort   int    `json:"discovery_port"`
	DiscoveryActive bool   `json:"discovery_active"`
	BeaconsSent     int64  `json:"beacons_sent"`
	BeaconsReceived int64  `json:"beacons_received"`
	LastBeaconMs    int64  `json:"last_beacon_ms"`
}

type peersResponse struct {
	Type  string `json:"type"`
	Peers []peer `json:"peers"`
}

type pongResponse struct {
	Type  string `json:"type"`
	Nonce string `json:"nonce"`
	TS    int64  `json:"ts"`
}

type knotCodeResponse struct {
	Type             string `json:"type"`
	Code             string `json:"code"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type knotResultResponse struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Peer  *peer  `json:"peer"`
	Error string `json:"error"`
}

type peerStatusResult struct {
	Type   string          `json:"type"`
	OK     bool            `json:"ok"`
	Status *statusResponse `json:"status"`
	Error  string          `json:"error"`
}

type sendFileResult struct {
	Type     string `json:"type"`
	OK       bool   `json:"ok"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
	Error    string `json:"error"`
}

type recvBeginResult struct {
	Type           string `json:"type"`
	OK             bool   `json:"ok"`
	Dir            string `json:"dir"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Error          string `json:"error"`
}

type recvWaitResult struct {
	Type       string `json:"type"`
	OK         bool   `json:"ok"`
	Path       string `json:"path"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
	FromNodeID string `json:"from_node_id"`
	Error      string `json:"error"`
}

type execRequestInfo struct {
	ExecID     string `json:"exec_id"`
	PeerID     string `json:"peer_id"`
	Command    string `json:"command"`
	Cwd        string `json:"cwd"`
	CreatedMs  int64  `json:"created_ms"`
	AgeSeconds int    `json:"age_seconds"`
}

type execRequestsResult struct {
	Type     string            `json:"type"`
	OK       bool              `json:"ok"`
	Requests []execRequestInfo `json:"requests"`
	Error    string            `json:"error"`
}

type execActionResult struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type runResult struct {
	Type     string `json:"type"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error"`
}

type clipboardBeginResult struct {
	Type             string `json:"type"`
	OK               bool   `json:"ok"`
	From             string `json:"from"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
	Error            string `json:"error"`
}

type clipboardStateResult struct {
	Type             string `json:"type"`
	OK               bool   `json:"ok"`
	Active           bool   `json:"active"`
	From             string `json:"from"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type clipboardResult struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Length int    `json:"length"`
	Error  string `json:"error"`
}

type daemonClient struct {
	client *proto.Client
}

func dialDaemon(g *globalFlags, timeout time.Duration) (*daemonClient, error) {
	var c, err = proto.DialTLS(g.addr(), true, timeout)
	if err != nil {
		return nil, fmt.Errorf("cannot reach weftlinkd at %s: %w (run 'weftlink up')", g.addr(), err)
	}
	return &daemonClient{client: c}, nil
}

func (d *daemonClient) close() {
	_ = d.client.Close()
}

func (d *daemonClient) call(req any, resp any, timeout time.Duration) error {
	if timeout <= 0 {
		return d.client.RoundTrip(req, resp)
	}
	return d.client.RoundTripTimeout(req, resp, timeout)
}

func fetchStatus(g *globalFlags, timeout time.Duration) (statusResponse, error) {
	var d, err = dialDaemon(g, timeout)
	if err != nil {
		return statusResponse{}, err
	}
	defer d.close()
	var resp statusResponse
	err = d.call(map[string]string{"type": "status"}, &resp, timeout)
	return resp, err
}

func printJSON(v any) error {
	var enc = json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
