// Command weftlink is the Weftlink command-line client.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var version = "0.4.0"

const usageText = `Weftlink command-line tool.

Usage:
  weftlink <command> [flags]

Commands:
  up                 Start weftlinkd if it is not already running
  status [peer]      Local status, or a paired peer's status
  ls                 List peers discovered on the LAN
  ping               Measure round-trip time to the local daemon
  knot [code]        Pair with a peer; without a code, print a new pair code
  send <peer> <file> Send a file to a paired peer
  recv [flags]       Wait for one incoming file (receiver must opt in first)
  run <peer> <cmd>   Request a remote command (peer must approve)
  requests           List pending remote command requests on this daemon
  approve <id>       Approve and run a pending remote command
  deny <id>          Reject a pending remote command
  clipboard ...      on/off/status/push/pull clipboard sync
  version            Print version information
  help               Show this help

Global flags:
  --host HOST        Daemon host (default 127.0.0.1, env WEFTLINK_HOST)
  --port PORT        Daemon port (default 7801, env WEFTLINK_PORT)
  --json             Emit machine-readable JSON
`

// globalFlags are accepted by every daemon-facing command.
type globalFlags struct {
	host string
	port int
	json bool
}

func (g *globalFlags) addr() string {
	return net.JoinHostPort(g.host, strconv.Itoa(g.port))
}

func newFlagSet(name string) (*flag.FlagSet, *globalFlags) {
	var fs = flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var g = &globalFlags{}
	fs.StringVar(&g.host, "host", envOr("WEFTLINK_HOST", "127.0.0.1"), "daemon host")
	fs.IntVar(&g.port, "port", envInt("WEFTLINK_PORT", 7801), "daemon port")
	fs.BoolVar(&g.json, "json", false, "emit JSON")
	return fs, g
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	var err = fs.Parse(reorderFlags(fs, args))
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	return err
}

// reorderFlags lets users write `weftlink knot CODE --peer host` as well as
// `weftlink knot --peer host CODE`. Go's flag package stops at the first
// positional argument, so move flags (and their values) to the front.
func reorderFlags(fs *flag.FlagSet, args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		var arg = args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		if strings.Contains(arg, "=") {
			continue
		}
		var name = strings.TrimLeft(arg, "-")
		if name == "" || name == "h" || name == "help" {
			continue
		}
		var f = fs.Lookup(name)
		if f == nil {
			continue
		}
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			continue
		}
		if i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return append(flags, positional...)
}

func noArgs(fs *flag.FlagSet) error {
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return nil
}

func envOr(key string, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	var raw = os.Getenv(key)
	if raw == "" {
		return def
	}
	var n, err = strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stdout)
		return
	}
	var cmd = os.Args[1]
	var args = os.Args[2:]

	var err error
	switch cmd {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
	case "version", "-v", "--version":
		err = cmdVersion(args)
	case "up":
		err = cmdUp(args)
	case "status":
		err = cmdStatus(args)
	case "ls":
		err = cmdLs(args)
	case "ping":
		err = cmdPing(args)
	case "knot":
		err = cmdKnot(args)
	case "send":
		err = cmdSend(args)
	case "recv":
		err = cmdRecv(args)
	case "run":
		err = cmdRun(args)
	case "requests":
		err = cmdRequests(args)
	case "approve":
		err = cmdApprove(args)
	case "deny":
		err = cmdDeny(args)
	case "clipboard":
		err = cmdClipboard(args)
	default:
		fmt.Fprintf(os.Stderr, "weftlink: unknown command %q\n\n", cmd)
		printUsage(os.Stderr)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "weftlink:", err)
		os.Exit(1)
	}
}
