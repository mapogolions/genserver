package tests

import (
	"errors"
	"net/rpc"
	"testing"

	"github.com/mapogolions/genserver"
	"github.com/stretchr/testify/assert"
)

func TestMathServer(t *testing.T) {
	t.Run("should add", func(t *testing.T) {
		// arrange
		s := NewMathServer()
		defer s.Close()

		// act
		call := s.Add(2)
		<-call.Done
		v, err := s.Value()

		// assert
		assert.Nil(t, err)
		assert.Equal(t, 2, v)
	})
}

func NewMathServer() *MathServer {
	return genserver.Listen(func(genserv genserver.GenServer) *MathServer {
		return &MathServer{GenServer: genserv}
	})
}

type MathServer struct {
	genserver.GenServer
	value int
}

func (s *MathServer) Add(v int) *rpc.Call {
	return s.Cast("+", v, nil, nil)
}

func (s *MathServer) Sub(v int) *rpc.Call {
	return s.Cast("+", v, nil, nil)
}

func (s *MathServer) Mul(v int) *rpc.Call {
	return s.Cast("*", v, nil, nil)
}

func (s *MathServer) Value() (int, error) {
	var v int
	err := s.Call("value", nil, &v)
	return v, err
}

func (s *MathServer) Handle(serviceMethod string, seq uint64, body any) (any, error) {
	var v any
	var err error
	switch serviceMethod {
	case "+":
		s.value += body.(int)
	case "-":
		s.value -= body.(int)
	case "*":
		s.value *= body.(int)
	case "value":
		v = s.value
	default:
		err = errors.New("unsupported operation")
	}
	return v, err
}
