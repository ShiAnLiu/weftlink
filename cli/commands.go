package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

func cmdVersion(args []string) error {
	var fs, g = newFlagSet("version")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if err := noArgs(fs); err != nil {
		return err
	}
	var s, daemonErr = fetchStatus(g, 2*time.Second)
	if g.json {
		var out = map[string]any{"version": version}
		if daemonErr == nil {
			out["daemon_version"] = s.Version
			out["node_id"] = s.NodeID
		}
		return printJSON(out)
	}
	fmt.Printf("weftlink %s\n", version)
	if daemonErr == nil {
		fmt.Printf("weftlinkd %s (node %s)\n", s.Version, s.NodeID)
	}
	return nil
}

func cmdStatus(args []string) error {
	var fs, g = newFlagSet("status")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("usage: weftlink status [peer]")
	}
	if fs.NArg() == 1 {
		var d, err = dialDaemon(g, 5*time.Second)
		if err != nil {
			return err
		}
		defer d.close()
		var result peerStatusResult
		err = d.call(map[string]string{
			"type": "peer_status",
			"peer": fs.Arg(0),
		}, &result, 15*time.Second)
		if err != nil {
			return err
		}
		if !result.OK || result.Status == nil {
			return fmt.Errorf("%s", result.Error)
		}
		if !g.json {
			fmt.Printf("remote peer: %s\n", fs.Arg(0))
		}
		return printStatus(*result.Status, g.json)
	}
	var s, err = fetchStatus(g, 5*time.Second)
	if err != nil {
		return err
	}
	return printStatus(s, g.json)
}

func printStatus(s statusResponse, jsonOut bool) error {
	if jsonOut {
		return printJSON(s)
	}
	fmt.Printf("weftlinkd %s - %s\n", s.Version, label(s.DisplayName))
	fmt.Printf("node id:     %s\n", s.NodeID)
	fmt.Printf("role:        %s\n", s.Role)
	fmt.Printf("fingerprint: %s\n", s.Fingerprint)
	fmt.Printf("peers:       %d discovered, %d paired\n", s.PeerCount, s.PairedCount)
	if s.DiscoveryActive {
		var last = "-"
		if s.LastBeaconMs > 0 {
			last = peerAge(s.LastBeaconMs)
		}
		fmt.Printf("discovery:   udp/%d active, sent %d, received %d (last %s)\n",
			s.DiscoveryPort, s.BeaconsSent, s.BeaconsReceived, last)
	} else {
		fmt.Printf("discovery:   udp/%d inactive\n", s.DiscoveryPort)
	}
	return nil
}

func cmdLs(args []string) error {
	var fs, g = newFlagSet("ls")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if err := noArgs(fs); err != nil {
		return err
	}
	var d, err = dialDaemon(g, 5*time.Second)
	if err != nil {
		return err
	}
	defer d.close()

	var resp peersResponse
	if err = d.call(map[string]string{"type": "peers"}, &resp, 5*time.Second); err != nil {
		return err
	}
	if g.json {
		return printJSON(resp.Peers)
	}
	if len(resp.Peers) == 0 {
		fmt.Println("no peers discovered (make sure another weftlinkd is running on the LAN)")
		return nil
	}
	var w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PAIRED\tNODE ID\tNAME\tADDRESS\tLAST SEEN")
	for _, p := range resp.Peers {
		var paired = "-"
		if p.Paired {
			paired = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			paired, p.PeerID, label(p.DisplayName), label(p.Address), peerAge(p.LastSeen))
	}
	return w.Flush()
}

func cmdPing(args []string) error {
	var fs, g = newFlagSet("ping")
	var count = fs.Int("count", 4, "number of requests")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if err := noArgs(fs); err != nil {
		return err
	}
	if *count <= 0 {
		return fmt.Errorf("count must be greater than zero")
	}
	var d, err = dialDaemon(g, 3*time.Second)
	if err != nil {
		return err
	}
	defer d.close()

	var rtts = make([]time.Duration, 0, *count)
	for i := 1; i <= *count; i++ {
		var start = time.Now()
		var resp pongResponse
		err = d.call(map[string]any{
			"type":  "ping",
			"nonce": strconv.Itoa(i),
			"ts":    start.UnixMilli(),
		}, &resp, 3*time.Second)
		if err != nil {
			return err
		}
		var rtt = time.Since(start)
		rtts = append(rtts, rtt)
		if !g.json {
			fmt.Printf("pong from %s: seq=%d time=%s\n", g.addr(), i, rtt.Round(time.Microsecond))
		}
	}

	var min, max, total = rtts[0], rtts[0], time.Duration(0)
	for _, rtt := range rtts {
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
		total += rtt
	}
	var avg = total / time.Duration(len(rtts))
	if g.json {
		return printJSON(map[string]any{
			"target": g.addr(),
			"sent":   len(rtts),
			"min_ms": float64(min.Microseconds()) / 1000,
			"avg_ms": float64(avg.Microseconds()) / 1000,
			"max_ms": float64(max.Microseconds()) / 1000,
		})
	}
	fmt.Printf("%d packets: min=%s avg=%s max=%s\n",
		len(rtts), min.Round(time.Microsecond), avg.Round(time.Microsecond), max.Round(time.Microsecond))
	return nil
}

func cmdUp(args []string) error {
	var fs, g = newFlagSet("up")
	var timeout = fs.Duration("timeout", 10*time.Second, "how long to wait for the daemon")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if err := noArgs(fs); err != nil {
		return err
	}
	if s, err := fetchStatus(g, 2*time.Second); err == nil {
		if !g.json {
			fmt.Println("weftlinkd is already running")
		}
		return printStatus(s, g.json)
	}

	if !isLocalHost(g.host) {
		return fmt.Errorf("'up' starts a local daemon; use --host 127.0.0.1 or start it on %s", g.host)
	}
	var path, err = findDaemon()
	if err != nil {
		return err
	}
	if err = spawnDaemon(path, g.port); err != nil {
		return err
	}
	var deadline = time.Now().Add(*timeout)
	for time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
		var s statusResponse
		s, err = fetchStatus(g, 2*time.Second)
		if err != nil {
			continue
		}
		if !g.json {
			fmt.Println("weftlinkd started; log:", daemonLogPath())
		}
		return printStatus(s, g.json)
	}
	return fmt.Errorf("weftlinkd did not become ready within %s (see %s)", *timeout, daemonLogPath())
}

func cmdKnot(args []string) error {
	var fs, g = newFlagSet("knot")
	var peerAddr = fs.String("peer", "", "remote peer ip[:port] (default: discovered peers)")
	var timeout = fs.Duration("timeout", 30*time.Second, "pairing timeout")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("usage: weftlink knot [code] [--peer host[:port]]")
	}

	var d, err = dialDaemon(g, 3*time.Second)
	if err != nil {
		return err
	}
	defer d.close()

	if fs.NArg() == 0 {
		var resp knotCodeResponse
		if err = d.call(map[string]string{"type": "knot_init"}, &resp, 3*time.Second); err != nil {
			return err
		}
		if g.json {
			return printJSON(resp)
		}
		fmt.Println("Pair code:", resp.Code)
		fmt.Printf("On the other device: weftlink knot %s\n", resp.Code)
		fmt.Printf("Code expires in %d seconds.\n", resp.ExpiresInSeconds)
		return nil
	}

	var code = strings.ToUpper(strings.TrimSpace(fs.Arg(0)))
	var resp knotResultResponse
	err = d.call(map[string]any{
		"type": "knot_join",
		"code": code,
		"host": *peerAddr,
	}, &resp, *timeout)
	if err != nil {
		return err
	}
	if g.json {
		var printErr = printJSON(resp)
		if printErr != nil {
			return printErr
		}
		if !resp.OK {
			return fmt.Errorf("pairing failed: %s", resp.Error)
		}
		return nil
	}
	if !resp.OK {
		return fmt.Errorf("pairing failed: %s", resp.Error)
	}
	if resp.Peer == nil {
		return fmt.Errorf("pairing returned no peer")
	}
	fmt.Printf("Paired with %s (%s) at %s\n",
		label(resp.Peer.DisplayName), resp.Peer.PeerID, label(resp.Peer.Address))
	return nil
}

func isLocalHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost":
		return true
	}
	var ip = net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func cmdSend(args []string) error {
	var fs, g = newFlagSet("send")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: weftlink send <peer> <file>")
	}
	var peerArg = fs.Arg(0)
	var source = fs.Arg(1)
	var abs, err = filepath.Abs(source)
	if err != nil {
		return err
	}
	var stat, statErr = os.Stat(abs)
	if statErr != nil {
		return statErr
	}
	if !stat.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", abs)
	}

	var d, dialErr = dialDaemon(g, 5*time.Second)
	if dialErr != nil {
		return dialErr
	}
	defer d.close()

	if !g.json {
		fmt.Printf("Sending %s (%s) to %s ...\n", filepath.Base(abs), humanBytes(stat.Size()), peerArg)
	}
	var start = time.Now()
	var result sendFileResult
	err = d.call(map[string]any{
		"type": "send_file",
		"peer": peerArg,
		"path": abs,
	}, &result, 30*time.Minute)
	if err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("send failed: %s", result.Error)
	}
	if g.json {
		return printJSON(result)
	}
	fmt.Printf("Sent %s (%s) to %s in %s\n", filepath.Base(abs),
		humanBytes(result.Size), peerArg, time.Since(start).Round(time.Millisecond))
	fmt.Printf("sha256: %s\n", result.Checksum)
	return nil
}

func cmdRecv(args []string) error {
	var fs, g = newFlagSet("recv")
	var dir = fs.String("dir", "", "destination directory (default ~/Downloads/Weftlink)")
	var from = fs.String("from", "", "only accept from this paired peer")
	var timeout = fs.Int("timeout", 300, "seconds to wait for a file")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if err := noArgs(fs); err != nil {
		return err
	}
	if *timeout < 10 {
		*timeout = 10
	}

	var d, err = dialDaemon(g, 5*time.Second)
	if err != nil {
		return err
	}
	defer d.close()

	var begin recvBeginResult
	err = d.call(map[string]any{
		"type":            "recv_begin",
		"dir":             *dir,
		"from":            *from,
		"timeout_seconds": *timeout,
	}, &begin, 15*time.Second)
	if err != nil {
		return err
	}
	if !begin.OK {
		return fmt.Errorf("%s", begin.Error)
	}
	if !g.json {
		var source = "any paired peer"
		if *from != "" {
			source = *from
		}
		fmt.Printf("Waiting for a file from %s in %s (up to %ds) ...\n",
			source, begin.Dir, begin.TimeoutSeconds)
	}

	var result recvWaitResult
	err = d.call(map[string]any{
		"type":            "recv_wait",
		"timeout_seconds": begin.TimeoutSeconds,
	}, &result, time.Duration(begin.TimeoutSeconds+20)*time.Second)
	if err != nil {
		return err
	}
	if g.json {
		if printErr := printJSON(result); printErr != nil {
			return printErr
		}
		if !result.OK {
			return fmt.Errorf("%s", result.Error)
		}
		return nil
	}
	if !result.OK {
		return fmt.Errorf("%s", result.Error)
	}
	fmt.Printf("Received %s (%s) from %s\n", result.Name,
		humanBytes(result.Size), label(result.FromNodeID))
	fmt.Printf("saved:  %s\n", result.Path)
	fmt.Printf("sha256: %s\n", result.Checksum)
	return nil
}

func humanBytes(n int64) string {
	const unit = int64(1024)
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	var divisor = unit
	var exponent = 0
	for value := n / unit; value >= unit; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(divisor), "KMGTPE"[exponent])
}

func label(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func peerAge(lastSeen int64) string {
	if lastSeen <= 0 {
		return "-"
	}
	var d = time.Since(time.UnixMilli(lastSeen))
	if d < 0 {
		d = 0
	}
	return d.Round(time.Second).String()
}
