package genserver

import (
	"errors"
	"net/rpc"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenServer(t *testing.T) {
	t.Run("should continue to process incoming requests if an error occurs during processing", func(t *testing.T) {
		// arrange
		expectedErr := errors.New("something went wrong")
		s := NewPanicServer(expectedErr)
		defer s.Close()

		// act
		err := s.Call("", nil, nil)

		// assert
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("should recover from panic when sending to closed mailbox of server process", func(t *testing.T) {
		// arrange
		s := NewEchoServer(1 * time.Hour)

		// act
		call1 := s.Cast("", "foo", nil, nil)
		go func() {
			time.Sleep(200 * time.Millisecond)
			s.Close()
		}()
		call2 := s.Cast("", "bar", nil, nil)
		<-call2.Done
		<-call1.Done

		// assert
		assert.ErrorIs(t, call1.Error, rpc.ErrShutdown)
		assert.NotNil(t, call2.Error)
		assert.Contains(t, call2.Error.Error(), "send on closed channel")
	})
}

var _ Behaviour = (*EchoServer)(nil)

func NewEchoServer(delay time.Duration) *EchoServer {
	genserv := newGenServer(0, 0)
	s := &EchoServer{GenServer: genserv, delay: delay}
	go genserv.Listen(s)
	return s
}

type EchoServer struct {
	GenServer
	delay time.Duration
}

func (s *EchoServer) Handle(serviceMethod string, _ uint64, body any) (any, error) {
	time.Sleep(s.delay)
	return body, nil
}

var _ Behaviour = (*PanicServer)(nil)

func NewPanicServer(err error) *PanicServer {
	return Listen(func(genserv GenServer) *PanicServer {
		return &PanicServer{GenServer: genserv, err: err}
	})
}

type PanicServer struct {
	GenServer
	err error
}

func (s *PanicServer) Handle(_ string, _ uint64, _ any) (any, error) {
	panic(s.err)
}
