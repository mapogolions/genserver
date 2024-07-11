package genserver

import (
	"io"
	"log"
	"net/rpc"
	"reflect"
)

type GenServerBehaviour interface {
	Handle(serviceMethod string, seq uint64, body any) (any, error)
}

type GenServer interface {
	rpc.ClientCodec
	Listen(GenServerBehaviour)
}

func NewGenServerAndListen[T GenServerBehaviour](f func(genserv GenServer) T) T {
	genserv := NewGenServer()
	c := f(genserv)
	go genserv.Listen(c)
	return c
}

func NewGenServer() *genServer {
	requests := make(chan request, 4096)
	responses := make(chan response, 4096)
	return &genServer{requests: requests, responses: responses}
}

type genServer struct {
	requests  chan request
	responses chan response
	current   response
}

func (c *genServer) WriteRequest(req *rpc.Request, body any) error {
	c.requests <- request{seq: req.Seq, serviceMethod: req.ServiceMethod, body: body}
	return nil
}

func (c *genServer) ReadResponseHeader(res *rpc.Response) error {
	kvRes, ok := <-c.responses
	if !ok {
		return io.EOF
	}
	c.current = kvRes
	res.Seq = kvRes.seq
	res.ServiceMethod = kvRes.serviceMethod
	if kvRes.result.Err != nil {
		res.Error = kvRes.result.Err.Error()
	}
	return kvRes.result.Err
}

func (c *genServer) ReadResponseBody(body any) error {
	// if `ReadResponseHeader` DOES NOT return error then `ReadResponseBody` will be called => c.current.result.Err == nil
	if c.current.result.Err != nil {
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

func (c *genServer) Listen(behaviour GenServerBehaviour) {
	for {
		req, ok := <-c.requests
		if !ok {
			// rpc.Client.Close -> codec.Close() -> close(codec.requestsStream)
			return
		}
		v, err := behaviour.Handle(req.serviceMethod, req.seq, req.body)
		c.responses <- response{seq: req.seq, serviceMethod: req.serviceMethod, result: result[any]{v, err}}
	}
}

/**
 * Codec's `Close` method called by the `rpc.Client`
 * `rpc.Client` provides the following guaranties:
 * - called once
 * - thread safety
 */
func (c *genServer) Close() error {
	close(c.requests)
	close(c.responses)
	return nil
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
	Err   error
}
