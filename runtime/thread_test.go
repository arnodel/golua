package runtime

import (
	"reflect"
	"testing"
)

func TestThread_Resume(t *testing.T) {
	type args struct {
		caller *Thread
		args   []Value
	}
	tests := []struct {
		name      string
		thread    *Thread
		args      args
		wantPanic interface{}
	}{
		{
			name: "caller must be running",
			thread: &Thread{
				status: ThreadSuspended,
			},
			args: args{
				caller: &Thread{
					status: ThreadDead,
				},
			},
			wantPanic: "Caller of thread to resume is not running",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := tt.thread
			gotPanic := func() (res interface{}) {
				defer func() { res = recover() }()
				_, _ = th.Resume(tt.args.caller, tt.args.args)
				return
			}()
			if !reflect.DeepEqual(gotPanic, tt.wantPanic) {
				t.Errorf("Thread.Resume() panic got %v, want %v", gotPanic, tt.wantPanic)
			}
		})
	}
}

func TestThread_Yield(t *testing.T) {
	type args struct {
		args []Value
	}
	tests := []struct {
		name      string
		thread    *Thread
		args      args
		wantPanic interface{}
	}{
		{
			name: "Thread to yield must be running",
			thread: &Thread{
				status: ThreadDead,
			},
			wantPanic: "Thread to yield is not running",
		},
		{
			name: "Caller of thread to yield must be OK",
			thread: &Thread{
				status: ThreadOK,
				caller: &Thread{
					status: ThreadDead,
				},
			},
			wantPanic: "Caller of thread to yield is not OK",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := tt.thread
			gotPanic := func() (res interface{}) {
				defer func() { res = recover() }()
				_, _ = th.Yield(tt.args.args)
				return
			}()
			if !reflect.DeepEqual(gotPanic, tt.wantPanic) {
				t.Errorf("Thread.Yield() panic got %v, want %v", gotPanic, tt.wantPanic)
			}
		})
	}
}

func TestThread_end(t *testing.T) {
	type args struct {
		args  []Value
		err   *Error
		extra interface{}
	}
	quotaErr := ContextTerminationError{message: "boo!"}
	tests := []struct {
		name      string
		thread    *Thread
		args      args
		wantPanic interface{}
	}{
		{
			name: "Thread to end must be running",
			thread: &Thread{
				status:   ThreadDead,
				caller:   &Thread{},
				resumeCh: make(chan valuesError),
			},
			wantPanic: "Called Thread.end on a non-running thread",
		},
		{
			name: "Caller of thread to end must be OK",
			thread: &Thread{
				status: ThreadOK,
				caller: &Thread{
					status: ThreadDead,
				},
				resumeCh: make(chan valuesError),
			},
			wantPanic: "Caller thread of ending thread is not OK",
		},
		{
			name: "Thread must not run out of resources",
			thread: &Thread{
				status: ThreadOK,
				caller: &Thread{
					resumeCh: make(chan valuesError, 1),
				},
				resumeCh: make(chan valuesError),
			},
			args: args{
				extra: quotaErr,
			},
			wantPanic: quotaErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := tt.thread
			th.Runtime = &Runtime{} // So releasing resources works.
			gotPanic := func() (res interface{}) {
				defer func() { res = recover() }()
				caller := th.caller // The caller is removed when th is killed
				th.end(tt.args.args, tt.args.err, tt.args.extra)
				_, _ = caller.getResumeValues()
				return
			}()
			if gotPanic != tt.wantPanic {
				t.Errorf("Thread.end() panic got %#v, want %#v", gotPanic, tt.wantPanic)
			}
		})
	}
}
