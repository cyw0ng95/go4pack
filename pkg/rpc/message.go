package rpc

// MessageType enumerates RPC message kinds.
type MessageType string

const (
	TypeRequest  MessageType = "req"
	TypeResponse MessageType = "res"
	TypeError    MessageType = "err"
	TypeEvent    MessageType = "evt" // optional server push / pub-sub
	TypeAck      MessageType = "ack" // ack for streaming chunks
)

// Header contains metadata for correlation and routing.
type Header struct {
	ID        string      `json:"id"`                // unique id per logical call (client generated)
	Type      MessageType `json:"type"`              // req | res | err | evt | ack
	Method    string      `json:"method,omitempty"`  // for requests
	Channel   string      `json:"channel,omitempty"` // subscription / event channel
	Seq       int64       `json:"seq,omitempty"`     // sequence for chunked or streamed messages
	Final     bool        `json:"final,omitempty"`   // marks final chunk / completion
	Timestamp int64       `json:"ts"`                // unix ms when produced
	Version   int         `json:"v"`                 // protocol version
}

// Message is the envelope passed over the wire. Payload uses generic any.
// Keep flat for efficient JSON / msgpack encoding.
type Message struct {
	Header  Header      `json:"header"`
	Payload interface{} `json:"payload,omitempty"` // request params / result / event body
	Error   *RPCError   `json:"error,omitempty"`   // filled if TypeError
}

// RPCError represents a structured wire error.
type RPCError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewRequest builds a request message.
func NewRequest(id, method string, payload interface{}, ts int64) Message {
	return Message{Header: Header{ID: id, Type: TypeRequest, Method: method, Timestamp: ts, Version: 1}, Payload: payload}
}

// NewResponse builds a response message.
func NewResponse(id string, payload interface{}, ts int64, final bool) Message {
	return Message{Header: Header{ID: id, Type: TypeResponse, Timestamp: ts, Version: 1, Final: final}, Payload: payload}
}

// NewError builds an error message.
func NewError(id string, code, msg string, data interface{}, ts int64) Message {
	return Message{Header: Header{ID: id, Type: TypeError, Timestamp: ts, Version: 1, Final: true}, Error: &RPCError{Code: code, Message: msg, Data: data}}
}

// NewEvent builds a server side event.
func NewEvent(channel string, seq int64, final bool, payload interface{}, ts int64) Message {
	return Message{Header: Header{ID: channel, Type: TypeEvent, Channel: channel, Seq: seq, Final: final, Timestamp: ts, Version: 1}, Payload: payload}
}

// NewAck builds an ack message for streaming reliability.
func NewAck(id string, seq int64, ts int64, final bool) Message {
	return Message{Header: Header{ID: id, Type: TypeAck, Seq: seq, Timestamp: ts, Version: 1, Final: final}}
}
