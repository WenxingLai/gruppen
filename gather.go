// Package gruppen provides a series of concurrent executors for functions.
package gruppen

import (
	"context"
	"golang.org/x/sync/errgroup"
)

// Executable represents a singular logic block.
type Executable func(ctx context.Context) func() (interface{}, error)

// Gather uses errgroup to execute fs and gather the results into a slice,
// the i-th item of which is the return value of fs[i] when no error occurs.
// All the funcs in fs will be executed and the first non-nil error will be returned, as it is for errgroup.
// Param limit is used to set the limit to the errgroup.
func Gather(ctx context.Context, limit int, fs []Executable) (
	[]interface{}, error) {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(limit)
	m := make([]interface{}, len(fs))
	withContexts := make([]func() (interface{}, error), 0, len(fs))
	for _, f := range fs {
		withContexts = append(withContexts, f(ctx))
	}
	for i, f := range withContexts {
		i, f := i, f
		wrap := func() error {
			ret, err := f()
			if err != nil {
				return err
			}
			m[i] = ret
			return nil
		}
		eg.Go(wrap)
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return m, nil
}

// GatherSoon is similar to Gather; it tries to stop execution when some error occurs.
func GatherSoon(ctx context.Context, limit int, fs []Executable) (
	[]interface{}, error) {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(limit)
	m := make([]interface{}, len(fs))
	hasErr := make(chan struct{}, len(fs))
	withContexts := make([]func() (interface{}, error), 0, len(fs))
	for _, f := range fs {
		withContexts = append(withContexts, f(ctx))
	}
	for i, f := range withContexts {
		i, f := i, f
		wrap := func() error {
			ret, err := f()
			if err != nil {
				hasErr <- struct{}{}
				return err
			}
			m[i] = ret
			return nil
		}
		stop := false
		select {
		case <-hasErr:
			stop = true
		default:
		}
		if stop {
			break
		}
		eg.Go(wrap)
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return m, nil
}
