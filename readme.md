**Worker Pool in Go 1.24**

- The pool is parametrized by generic resource type
- Resource must implement io.Closer.
- The pool accepts a creator function that returns a resource implementing `io.Closer`.
- The pool accepts a context, allowing workers to detect cancellation signals.
- The pool creates the specified number of workers in its constructor and passes context and creator function as the arguments
- Clients submit tasks (functions) to a channel.
- Each worker creates its resource on startup and defers closing it until exit.
- Each worker consumes tasks from the channel and executes them, passing the resource as an argument.