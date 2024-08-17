package genserver

import (
	"net/rpc"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenServer(t *testing.T) {
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
