// Package jsonrpc implements a JSON-RPC ClientCodec and ServerCodec for the birpc package.
//
// Beside struct types, JSONCodec allows using positional arguments.
// Use []interface{} as the type of argument when sending and receiving methods.
//
// Positional arguments example:
// 	server.Handle("add", func(client *birpc.Client, args []interface{}, result *float64) error {
// 		*result = args[0].(float64) + args[1].(float64)
// 		return nil
// 	})
//
//	var result float64
// 	client.Call("add", []interface{}{1, 2}, &result)
//
package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/cgrates/rpc"
)

type jsonCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	msg            message
	serverRequest  serverRequest
	clientResponse clientResponse

	// JSON-RPC clients can use arbitrary json values as request IDs.
	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	mutex   sync.Mutex // protects seq, pending
	pending map[uint64]*json.RawMessage
	seq     uint64
}

// NewJSONBirpcCodec returns a new birpc.Codec using JSON-RPC on conn.
func NewJSONBirpcCodec(conn io.ReadWriteCloser) rpc.BirpcCodec {
	return &jsonCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		pending: make(map[uint64]*json.RawMessage),
	}
}

// serverRequest and clientResponse combined
type message struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
	Id     *json.RawMessage `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
}

func (c *jsonCodec) ReadHeader(req *rpc.Request, resp *rpc.Response) error {
	c.msg = message{}
	if err := c.dec.Decode(&c.msg); err != nil {
		return err
	}

	if c.msg.Method != "" {
		// request comes to server
		c.serverRequest.Id = c.msg.Id
		c.serverRequest.Method = c.msg.Method
		c.serverRequest.Params = c.msg.Params

		req.ServiceMethod = c.serverRequest.Method

		// JSON request id can be any JSON value;
		// RPC package expects uint64.  Translate to
		// internal uint64 and save JSON on the side.
		if c.serverRequest.Id == nil {
			// Notification
		} else {
			c.mutex.Lock()
			c.seq++
			c.pending[c.seq] = c.serverRequest.Id
			c.serverRequest.Id = nil
			req.Seq = c.seq
			c.mutex.Unlock()
		}
	} else {
		// response comes to client
		err := json.Unmarshal(*c.msg.Id, &c.clientResponse.Id)
		if err != nil {
			return err
		}
		c.clientResponse.Result = c.msg.Result
		c.clientResponse.Error = c.msg.Error

		resp.Error = ""
		resp.Seq = c.clientResponse.Id
		if c.clientResponse.Error != nil || c.clientResponse.Result == nil {
			x, ok := c.clientResponse.Error.(string)
			if !ok {
				return fmt.Errorf("invalid error %v", c.clientResponse.Error)
			}
			if x == "" {
				x = "unspecified error"
			}
			resp.Error = x
		}
	}
	return nil
}

func (c *jsonCodec) ReadRequestBody(x interface{}) error {
	if x == nil {
		return nil
	}
	if c.serverRequest.Params == nil {
		return errMissingParams
	}
	// JSON params is array value.
	// RPC params is struct.
	// Unmarshal into array containing struct for now.
	// Should think about making RPC more general.
	var params [1]interface{}
	params[0] = x
	return json.Unmarshal(*c.serverRequest.Params, &params)
}

func (c *jsonCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	return json.Unmarshal(*c.clientResponse.Result, x)
}

func (c *jsonCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	return c.enc.Encode(&clientRequest{
		Method: r.ServiceMethod,
		Params: [1]interface{}{param},
		Id:     r.Seq,
	})
}

func (c *jsonCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	c.mutex.Lock()
	b, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	if b == nil {
		// Invalid request so no id.  Use JSON null.
		b = &null
	}
	resp := serverResponse{Id: b}
	if r.Error == "" {
		resp.Result = x
	} else {
		resp.Error = r.Error
	}
	return c.enc.Encode(resp)
}

func (c *jsonCodec) Close() error {
	return c.c.Close()
}
