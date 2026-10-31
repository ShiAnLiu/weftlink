Weftlink - quick start (Linux / macOS)
======================================

This archive contains two binaries:

  weftlinkd   the daemon (long-running node)
  weftlink    the command-line client

They must stay in the same directory: `weftlink up` starts the
`weftlinkd` found next to itself.

First run
---------
  chmod +x weftlink weftlinkd
  ./weftlink up            # starts the daemon (TLS on :7801)
  ./weftlink status        # node id, role, fingerprint, peer counts
  ./weftlink ping          # round-trip latency to the local daemon

Daemon state lives in ~/.weftlink (override with WEFTLINK_HOME):
  ~/.weftlink/knots/                         identity keys (Ed25519)
  ~/.weftlink/weftlinkd-cert.pem, -key.pem   TLS cert (auto-generated)
  ~/.weftlink/weftlinkd.log                  daemon log

LAN pairing (two machines on the same network)
----------------------------------------------
  machine A:  ./weftlink knot              # prints a 6-char code (2 min TTL)
  machine B:  ./weftlink knot <CODE>
              # if discovery fails, point at A explicitly:
              ./weftlink knot <CODE> --peer 192.168.x.x:7801
  both:       ./weftlink ls                # paired peers are marked with *

Troubleshooting
---------------
  `ls` shows nothing
      - both machines on the same subnet / same Wi-Fi
      - inbound TCP 7801 and UDP 7802 allowed by the firewall
      - or pair explicitly with --peer <IP>:7801
  `up` says weftlinkd not found
      - both files must be in the same directory, named exactly so
  stopping the daemon
      - pkill weftlinkd

Security notes
--------------
- Transport is TLS 1.3. The certificate is self-signed and generated on
  first start; clients pin it, they do not use the system CA store.
- Pairing exchanges public-key fingerprints only; private keys never
  leave the machine and are never logged.
