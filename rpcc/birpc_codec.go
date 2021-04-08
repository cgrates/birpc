package rpcc

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// A Codec implements reading and writing of RPC requests and responses.
// The client calls ReadHeader to read a message header.
// The implementation must populate either Request or Response argument.
// Depending on which argument is populated, ReadRequestBody or
// ReadResponseBody is called right after ReadHeader.
// ReadRequestBody and ReadResponseBody may be called with a nil
// argument to force the body to be read and then discarded.
type BirpcCodec interface {
	// ReadHeader must read a message and populate either the request
	// or the response by inspecting the incoming message.
	ReadHeader(*Request, *Response) error

	// ReadRequestBody into args argument of handler function.
	ReadRequestBody(interface{}) error

	// ReadResponseBody into reply argument of handler function.
	ReadResponseBody(interface{}) error

	// WriteRequest must be safe for concurrent use by multiple goroutines.
	WriteRequest(*Request, interface{}) error

	// WriteResponse must be safe for concurrent use by multiple goroutines.
	WriteResponse(*Response, interface{}) error

	// Close is called when client/server finished with the connection.
	Close() error
}

type gobCodec struct {
	rwc    io.ReadWriteCloser
	dec    *gob.Decoder
	enc    *gob.Encoder
	encBuf *bufio.Writer
}

type message struct {
	Seq           uint64
	ServiceMethod string
	Error         string
}

// NewGobCodec returns a new biCodec using gob encoding/decoding on conn.
func NewGobBirpcCodec(conn io.ReadWriteCloser) BirpcCodec {
	buf := bufio.NewWriter(conn)
	return &gobCodec{
		rwc:    conn,
		dec:    gob.NewDecoder(conn),
		enc:    gob.NewEncoder(conn),
		encBuf: buf,
	}
}

func (c *gobCodec) ReadHeader(req *Request, resp *Response) error {
	var msg message
	if err := c.dec.Decode(&msg); err != nil {
		return err
	}

	if msg.ServiceMethod != "" {
		req.Seq = msg.Seq
		req.ServiceMethod = msg.ServiceMethod
	} else {
		resp.Seq = msg.Seq
		resp.Error = msg.Error
	}
	return nil
}

func (c *gobCodec) ReadRequestBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *gobCodec) ReadResponseBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *gobCodec) WriteRequest(r *Request, body interface{}) (err error) {
	if err = c.enc.Encode(r); err != nil {
		return
	}
	if err = c.enc.Encode(body); err != nil {
		return
	}
	return c.encBuf.Flush()
}

func (c *gobCodec) WriteResponse(r *Response, body interface{}) (err error) {
	if err = c.enc.Encode(r); err != nil {
		if c.encBuf.Flush() == nil {
			// Gob couldn't encode the header. Should not happen, so if it does,
			// shut down the connection to signal that the connection is broken.
			log.Println("rpc: gob error encoding response:", err)
			c.Close()
		}
		return
	}
	if err = c.enc.Encode(body); err != nil {
		if c.encBuf.Flush() == nil {
			// Was a gob problem encoding the body but the header has been written.
			// Shut down the connection to signal that the connection is broken.
			log.Println("rpc: gob error encoding body:", err)
			c.Close()
		}
		return
	}
	return c.encBuf.Flush()
}

func (c *gobCodec) Close() error {
	return c.rwc.Close()
}
