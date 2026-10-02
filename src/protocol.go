// Protocol: Weft Protocol message types and framing
package main

type MsgType int

const (
	MsgHello           MsgType = 0
	MsgPairRequest     MsgType = 1
	MsgPairConfirm     MsgType = 2
	MsgPing            MsgType = 3
	MsgFileOffer       MsgType = 4
	MsgFileChunk       MsgType = 5
	MsgClipboardUpdate MsgType = 6
	MsgExecRequest     MsgType = 7
)

type HelloMsg struct {
	node  NodeInfo
	nonce string
}

type PairRequestMsg struct {
	pairCode string
	node     NodeInfo
}

type PairConfirmMsg struct {
	node        NodeInfo
	fingerprint string
}

type PingMsg struct {
	nonce string
	ts    int64
}

type FileOfferMsg struct {
	fileID string
	name   string
	size   int64
	mime   string
}

type FileChunkMsg struct {
	fileID string
	offset int64
	data   []byte
	final  bool
}

type ClipboardUpdateMsg struct {
	text string
	mime string
	ts   int64
}

type ExecRequestMsg struct {
	execID  string
	command string
	cwd     string
	timeout int
}
