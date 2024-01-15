package gruppen

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

var theError = errors.New("error occurs")

type args struct {
	ctx   context.Context
	limit int
	fs    []Executable
}

type testCase struct {
	name    string
	args    args
	want    []interface{}
	wantErr bool
}

func wrap(f func() (interface{}, error)) Executable {
	return func(ctx context.Context) func() (interface{}, error) {
		return func() (interface{}, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return f()
		}
	}
}

func okAfterWait(s interface{}) Executable {
	return wrap(
		func() (interface{}, error) {
			d := 5 * time.Millisecond
			time.Sleep(d)
			return []interface{}{s}, nil
		},
	)
}

func errAfterWait() Executable {
	return wrap(
		func() (interface{}, error) {
			d := 5 * time.Millisecond
			time.Sleep(d)
			return nil, theError
		},
	)
}

func errNow() Executable {
	return func(ctx context.Context) func() (interface{}, error) {
		return func() (interface{}, error) {
			return nil, theError
		}
	}
}

func someStrings() ([]Executable, []interface{}) {
	fs := make([]Executable, 0, 100)
	want := make([]interface{}, 0, 100)
	for i := 0; i < 100; i++ {
		si := strconv.Itoa(i)
		fs = append(fs, okAfterWait(si))
		want = append(want, []interface{}{si})
	}
	return fs, want
}

func allStringGood() testCase {
	fs, want := someStrings()
	return testCase{
		name: "allStringGood",
		args: args{
			ctx:   context.Background(),
			limit: 10,
			fs:    fs,
		},
		want:    want,
		wantErr: false,
	}
}

func intAndStringGood() testCase {
	return testCase{
		name: "intAndStringGood",
		args: args{
			ctx:   context.Background(),
			limit: 10,
			fs: []Executable{
				okAfterWait("0"), okAfterWait(1),
			},
		},
		want:    []interface{}{[]interface{}{"0"}, []interface{}{1}},
		wantErr: false,
	}
}

func allStringErr() testCase {
	fs, _ := someStrings()
	fs[11] = errAfterWait()
	return testCase{
		name: "allStringErr",
		args: args{
			ctx:   context.Background(),
			limit: 10,
			fs:    fs,
		},
		want:    nil,
		wantErr: true,
	}
}

func allErr() testCase {
	fs := make([]Executable, 0, 100)
	for i := 0; i < 100; i++ {
		fs = append(fs, errAfterWait())
	}
	return testCase{
		name: "allErr",
		args: args{
			ctx:   context.Background(),
			limit: 10,
			fs:    fs,
		},
		want:    nil,
		wantErr: true,
	}
}

var falsifiedExecutionNum = atomic.Uint32{}

const falsifiedTotal = 100

func falsified(context.Context) func() (interface{}, error) {
	return func() (interface{}, error) {
		falsifiedExecutionNum.Add(1)
		return nil, nil
	}
}

func errStopsExecution() testCase {
	fs := []Executable{
		wrap(
			func() (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return nil, nil
			},
		),
		wrap(
			func() (interface{}, error) {
				time.Sleep(5 * time.Millisecond)
				return nil, nil
			},
		),
		errNow(),
	}
	for i := 0; i < falsifiedTotal; i++ {
		fs = append(fs, falsified)
	}
	return testCase{
		name: "errStopsExecution",
		args: args{
			ctx:   context.Background(),
			limit: 2,
			fs:    fs,
		},
		want:    nil,
		wantErr: true,
	}
}

func TestGather(t *testing.T) {
	tests := []testCase{
		allStringGood(),
		intAndStringGood(),
		allStringErr(),
		allErr(),
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := Gather(tt.args.ctx, tt.args.limit, tt.args.fs)
				if (err != nil) != tt.wantErr {
					t.Errorf("Gather() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Gather() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestGatherSoon(t *testing.T) {
	tests := []testCase{
		allStringGood(),
		intAndStringGood(),
		allStringErr(),
		allErr(),
		errStopsExecution(),
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := GatherSoon(tt.args.ctx, tt.args.limit, tt.args.fs)
				if (err != nil) != tt.wantErr {
					t.Errorf("Gather() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Gather() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
	if falsifiedExecutionNum.Load() == falsifiedTotal {
		t.Errorf("falsifiedExecutionNum == falsifiedTotal, errStopsExecution failed")
	}
	t.Logf("falsifiedExecutionNum: %d", falsifiedExecutionNum.Load())
}
