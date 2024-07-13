## Gen Server

**gen_server** in Erlang represents a set of abstractions and concrete implementations of parts of code necessary for writing *server processes*.

**Server process** is an informal name for a dedicated (spawned) concurrency unit that runs for an extended period and listens for incoming requests from other concurrency unit.


In Erlang, the unit of concurrency is the lightweight process. These processes do not share memory and communicate using *asynchronous message passing*. In Go, the unit of concurrency is the *goroutine*. To reproduce asynchronous message communication in Go, this project uses *buffered channels*.

### Example

Let's assume that your app requires some kind of in memory storage. It must support simultaneous access by multiple concurrency units (i.e., be thread-safe).

#### Shared Memory & Locks

One possible solution is to use a shared memory model by writing a structure that utilizes a *hash table* and a concurrency primitive such as a *read-write lock*.

#### Message passing

At the same time, you can look at the task from another perspective. For example, in Erlang, where you do not have access to shared memory, a primary solution is to create a separate concurrency unit that handles requests, modifies its internal state, and sends responses. This corresponds to what is described above as a *server proces*.

#### How to create a *server process*

1) define a server that embeds `genserver.GenServer`.

```golang
type EventLogger struct {
    genserver.GenServer

    // define state
}
```

2) implement the `genserver.GenServerBehaviour`  contract

```golang
func (s *EventLogger) Handle(serviceMethod string, seq uint64, body any) (any, error) {
    panic("not implemented") 
}
```

3) write a factory function

```golang
func NewLogServer(/* state */) *EventLogger {
	return Listen(func(genserv GenServer) *EventLogger {
		return &EventLogger{GenServer: genserv, /* state */ }
	})
}
```