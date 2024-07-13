## Gen Server

**gen_server** module in Erlang indeed represents a set of abstractions and concrete implementations of common parts of code necessary for writing *server processes*.


**Server process** is an informal name for a dedicated (spawned) concurrency unit that runs for an extended period and listens for incoming requests from other concurrency unit.


In Erlang, the unit of concurrency is the lightweight process. These processes do not share memory and communicate using asynchronous message passing. In Go, the unit of concurrency is the goroutine. To reproduce asynchronous message communication in Go, this project uses buffered channels.


Let's assume that the project requires some kind of storage. It doesn't matter what kind—settings, sessions, etc.

Requirements:

- DOES NOT need to be persistent (the lifetime of the data is limited to the lifetime of the application itself).
- MUST support simultaneous access by multiple concurrency units (thread-safety).

One possible solution is to use a shared memory model by writing a structure that utilizes a `map` and `rwlock`.

At the same time, you can look at the task from another perspective, considering programming languages that prefer different models. For example, in Erlang, a primary solution would be to create a separate concurrency unit that handles requests, modifies internal state, and sends responses. This corresponds to what is described above as a *server proces*.


to be continued ...