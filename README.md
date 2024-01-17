# Gruppen

This repository provides Go concurrency utils to execute tasks and collect results or errors concurrently.

`gruppen.Gather` and `gruppen.GatherSoon` use `errgroup.Group` to control and limit concurrency, 
the results are collected to a `[]interface{}`. If any non-nil error occurs, the first one would be returned.

When a non-nil error is found, `gruppen.GatherSoon` uses an additional channel to cancel all the tasks not started;
while `gruppen.Gather` continues to execute the remaining tasks with the cancel func for the `ctx` called.

`gruppen` comes from the name of a well-known 20-th century [composition](https://en.wikipedia.org/wiki/Gruppen) for three orchestras by Karlheinz Stockhausen.

## Comparison

[`facebookgo/errgroup`](https://pkg.go.dev/github.com/facebookgo/errgroup) is similar to `sync.WaitGroup`, 
but it collects all the errors.

[`gollback`](https://github.com/vardius/gollback) and [`hunch`](https://github.com/AaronJan/Hunch)
provide functions as `gruppen.Gather` does, but the concurrency cannot be set directly.
