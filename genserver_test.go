package genserver

import (
	"errors"
	"net/rpc"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestKVStoreCodec(t *testing.T) {
	t.Run("should return shutdown error when trying to make rpc call on closed client", func(t *testing.T) {
		// arrange
		dict := NewDict[string, int]()
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act
		<-time.After(200 * time.Millisecond) // give a chance to start goroutine to listen
		client.Close()
		var reply int
		call := client.Go("get", "foo", &reply, nil)

		// assert
		assert.Equal(t, 0, reply)
		assert.ErrorIs(t, call.Error, rpc.ErrShutdown)
	})

	t.Run("delete key should return error if key does not exists", func(t *testing.T) {
		// arrange
		dict := NewDict[string, int]()
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act
		var reply int
		call := client.Go("delete", "one", &reply, nil)
		<-call.Done

		// assert
		assert.NotNil(t, call.Error)
		assert.Equal(t, 0, reply)
	})

	t.Run("should delete key from store if it exists", func(t *testing.T) {
		// arrange
		dict := NewDict[string, int](KeyValuePair[string, int]{"one", -1})
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act
		var reply int
		call := client.Go("delete", "one", &reply, nil)
		<-call.Done

		// assert
		assert.Nil(t, call.Error)
		assert.Equal(t, -1, reply)
	})

	t.Run("should put key value pair into kvstore", func(t *testing.T) {
		// arrange
		dict := NewDict[string, int]()
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act + assert
		err := client.Call("put", KeyValuePair[string, int]{"one", -1}, nil)
		assert.Nil(t, err)

		v, err := dict.Get("one") // check internal state of the store
		assert.Nil(t, err)
		assert.Equal(t, -1, v)

		var actual int // get key using rpc client
		err = client.Call("get", "one", &actual)
		assert.Nil(t, err)
		assert.Equal(t, -1, actual)
	})

	t.Run("should get value by key from kvstore using blocking api of rpc-client", func(t *testing.T) {
		// arrange
		dict := NewDict(KeyValuePair[string, int]{"one", -1})
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act
		var actual int
		err := client.Call("get", "one", &actual)

		// assert
		assert.Nil(t, err)
		assert.Equal(t, -1, actual)
	})

	t.Run("should ignore that reply is not pointer", func(t *testing.T) {
		// arrange
		dict := NewDict(KeyValuePair[string, int]{"one", -1})
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act
		var reply int
		call := client.Go("get", "one", reply, nil)
		<-call.Done

		// assert
		assert.Equal(t, 0, reply)
	})

	t.Run("should ignore wrong type of reply", func(t *testing.T) {
		// arrange
		dict := NewDict(KeyValuePair[string, int]{"one", -1})
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act
		var reply string
		call := client.Go("get", "one", &reply, nil)
		<-call.Done

		// assert
		assert.Empty(t, reply)
	})

	t.Run("should ignore nil reply", func(t *testing.T) {
		// arrange
		dict := NewDict(KeyValuePair[string, int]{"one", -1})
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act + assert
		call := client.Go("get", "one", nil, nil)
		<-call.Done
	})

	t.Run("should get value by key from store using non-blocking api of rpc-client", func(t *testing.T) {
		// arrange
		dict := NewDict(KeyValuePair[string, int]{"one", -1})
		codec := NewKVStoreCodec[string, int](dict)
		client := rpc.NewClientWithCodec(codec)

		// act
		var actual int
		call := client.Go("get", "one", &actual, nil)
		<-call.Done

		// assert
		assert.Equal(t, -1, actual)
	})
}

type KVStore[K comparable, V any] interface {
	Get(key K) (V, error)
	Put(key K, v V) error
	Delete(key K) (V, error)
}

type KeyValuePair[K, V any] struct {
	Key   K
	Value V
}

type kvStoreCodec[K comparable, V any] struct {
	GenServer
	store KVStore[K, V]
}

// // version 1
// func NewKVStoreCodec[K comparable, V any](store KVStore[K, V]) *kvStoreCodec[K, V] {
// 	c := &kvStoreCodec[K, V]{store: store, GenServer: NewGenServer()}
// 	go c.Listen(c)
// 	return c
// }

// version 2
func NewKVStoreCodec[K comparable, V any](store KVStore[K, V]) *kvStoreCodec[K, V] {
	return NewGenServerAndListen(func(genserv GenServer) *kvStoreCodec[K, V] {
		return &kvStoreCodec[K, V]{store: store, GenServer: genserv}
	})
}

func (c *kvStoreCodec[K, V]) Handle(serviceMethod string, _ uint64, body any) (any, error) {
	var v any
	var err error
	switch serviceMethod {
	case "get":
		v, err = c.store.Get(body.(K))
	case "delete":
		v, err = c.store.Delete(body.(K))
	case "put":
		kvp, ok := body.(KeyValuePair[K, V])
		if ok {
			err = c.store.Put(kvp.Key, kvp.Value)
		} else {
			err = errors.New("invalid arguments")
		}
	default:
		panic("not implemented")
	}
	return v, err
}

type dict[K comparable, V any] struct {
	data map[K]V
}

func NewDict[K comparable, V any](pairs ...KeyValuePair[K, V]) dict[K, V] {
	data := make(map[K]V)
	if len(pairs) > 0 {
		for _, pair := range pairs {
			data[pair.Key] = pair.Value
		}
	}
	return dict[K, V]{data: data}
}

func (d dict[K, V]) Get(key K) (V, error) {
	v, ok := d.data[key]
	if !ok {
		return v, errors.New("not found")
	}
	return v, nil
}

func (d dict[K, V]) Put(key K, value V) error {
	_, ok := d.data[key]
	if ok {
		return errors.New("key already exists")
	}
	d.data[key] = value
	return nil
}

func (d dict[K, V]) Delete(key K) (V, error) {
	v, ok := d.data[key]
	if !ok {
		return v, errors.New("key does not exist")
	}
	return v, nil
}
