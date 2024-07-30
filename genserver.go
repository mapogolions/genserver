package genserver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"reflect"
)

func Reply[T any](call *rpc.Call) T {
	return *(call.Reply.(*T))
}

type Behaviour interface {
	Handle(serviceMethod string, seq uint64, body any) (any, error)
}

type GenServer interface {
	Listen(Behaviour)
	Cast(serviceMethod string, args any, reply any, done chan *rpc.Call) *rpc.Call
	Call(serviceMethod string, args any, reply any) error
	Close() error
}

func Listen[T Behaviour](f func(GenServer) T) T {
	serv := NewGenServer()
	behaviour := f(serv)
	go serv.Listen(behaviour)
	return behaviour
}

func NewGenServer() *genServer {
	return newGenServer(4096, 4096)
}

func newGenServer(incap uint, outcap uint) *genServer {
	requests := make(chan request, incap)
	responses := make(chan response, outcap)
	codec := &genServerCodec{requests: requests, responses: responses}
	client := rpc.NewClientWithCodec(codec)
	return &genServer{codec: codec, client: client}
}

type genServer struct {
	codec  *genServerCodec
	client *rpc.Client
}

// Implement `GenServer`
func (s *genServer) Cast(serviceMethod string, args any, reply any, done chan *rpc.Call) *rpc.Call {
	return s.client.Go(serviceMethod, args, reply, done)
}

func (s *genServer) Call(serviceMethod string, args any, reply any) error {
	return s.client.Call(serviceMethod, args, reply)
}

func (s *genServer) Close() error {
	return s.client.Close()
}

func (s *genServer) Listen(behaviour Behaviour) {
	s.codec.Listen(behaviour)
}

type genServerCodec struct {
	requests  chan request
	responses chan response
	current   response
}

// Implement `rpc.ClientCodec`
func (c *genServerCodec) WriteRequest(req *rpc.Request, body any) error {
	var err error
	tryCatch(func() {
		c.requests <- request{seq: req.Seq, serviceMethod: req.ServiceMethod, body: body}
	}, &err)
	return err
}

func (c *genServerCodec) ReadResponseHeader(res *rpc.Response) error {
	response, ok := <-c.responses
	if !ok {
		return io.EOF
	}
	c.current = response
	res.Seq = response.seq
	res.ServiceMethod = response.serviceMethod
	if response.result.Error != nil {
		res.Error = response.result.Error.Error()
	}
	return response.result.Error
}

func (c *genServerCodec) ReadResponseBody(body any) error {
	// if `ReadResponseHeader` DOES NOT return error then `ReadResponseBody` will be called => c.current.result.Err == nil
	if c.current.result.Error != nil {
		log.Fatal("must be unreachable")
	}
	v := c.current.result.Value
	if v == nil {
		return nil
	}
	if body == nil { // ignore nil `reply`
		return nil
	}
	tbody := reflect.TypeOf(body)
	if tbody.Kind() != reflect.Pointer { // should ignore if `reply` non-pointer type
		return nil
	}
	if tbody.Elem() != reflect.TypeOf(v) { // should ignore if `reply` has wrong type
		return nil
	}
	vbody := reflect.ValueOf(body)
	vbody.Elem().Set(reflect.ValueOf(v))
	return nil
}

/**
 * Codec's `Close` method called by the `rpc.Client`
 * `rpc.Client` provides the following guaranties:
 * - called once
 * - thread safety
 */
func (c *genServerCodec) Close() error {
	close(c.requests)
	close(c.responses)
	return nil
}

// It's not part of `rpc.ClientCodec`
func (c *genServerCodec) Listen(behaviour Behaviour) {
	for {
		req, ok := <-c.requests
		if !ok {
			// rpc.Client.Close -> codec.Close() -> close(codec.requestsStream)
			return
		}
		v, err := behaviour.Handle(req.serviceMethod, req.seq, req.body)
		var crucialErr error
		tryCatch(func() {
			c.responses <- response{
				seq:           req.seq,
				serviceMethod: req.serviceMethod,
				result:        result[any]{Value: v, Error: err},
			}
		}, &crucialErr)
		if crucialErr != nil {
			log.Print(crucialErr)
		}
	}
}

type request struct {
	seq           uint64
	serviceMethod string
	body          any
}

type response struct {
	seq           uint64
	serviceMethod string
	result        result[any]
}

type result[T any] struct {
	Value T
	Error error
}

func tryCatch(f func(), crucialErr *error) {
	defer func() {
		if info := recover(); info != nil {
			if errorMessage, ok := info.(string); ok {
				*crucialErr = errors.New(errorMessage)
				return
			}
			if err, ok := info.(error); ok {
				*crucialErr = err
				return
			}
			*crucialErr = fmt.Errorf("%v", info)
		}
	}()
	f()
}
