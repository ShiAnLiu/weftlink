package proto

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"time"
)

// DefaultTimeout bounds a single request/response round trip.
const DefaultTimeout = 5 * time.Second

// Client speaks framed Weft Protocol over an existing connection.
type Client struct {
	conn net.Conn
}

// DialTLS connects to addr and returns a framed client.
//
// insecure skips certificate verification. Weftlink daemons use self-signed
// certificates and authenticate peers at the application layer, so callers on
// the loopback/LAN path pass true during the current milestone.
func DialTLS(addr string, insecure bool, timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	var cfg = &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec // self-signed daemon certs (M1)
	var dialer = &net.Dialer{Timeout: timeout}
	var conn net.Conn
	var err error
	conn, err = tls.DialWithDialer(dialer, "tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// NewClient wraps an established connection.
func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// RoundTrip sends req and decodes the next response frame into resp.
func (c *Client) RoundTrip(req any, resp any) error {
	return c.RoundTripTimeout(req, resp, DefaultTimeout)
}

// RoundTripTimeout is RoundTrip with an explicit deadline.
func (c *Client) RoundTripTimeout(req any, resp any, timeout time.Duration) error {
	if err := c.SendTimeout(req, timeout); err != nil {
		return err
	}
	return c.ReceiveTimeout(resp, timeout)
}

// SendTimeout encodes value and writes one frame with a write deadline.
func (c *Client) SendTimeout(value any, timeout time.Duration) error {
	var payload []byte
	var err error
	payload, err = json.Marshal(value)
	if err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if err = c.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	return WriteFrame(c.conn, payload)
}

// ReceiveTimeout reads one frame and decodes it with a read deadline.
func (c *Client) ReceiveTimeout(value any, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	var raw []byte
	if err := ReadFrame(c.conn, &raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}
