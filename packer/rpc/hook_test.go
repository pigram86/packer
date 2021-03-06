package rpc

import (
	"cgl.tideland.biz/asserts"
	"github.com/mitchellh/packer/packer"
	"net/rpc"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestHookRPC(t *testing.T) {
	assert := asserts.NewTestingAsserts(t, true)

	// Create the UI to test
	h := new(packer.MockHook)

	// Serve
	server := rpc.NewServer()
	RegisterHook(server, h)
	address := serveSingleConn(server)

	// Create the client over RPC and run some methods to verify it works
	client, err := rpc.Dial("tcp", address)
	assert.Nil(err, "should be able to connect")

	hClient := Hook(client)

	// Test Run
	ui := &testUi{}
	hClient.Run("foo", ui, nil, 42)
	assert.True(h.RunCalled, "run should be called")

	// Test Cancel
	hClient.Cancel()
	assert.True(h.CancelCalled, "cancel should be called")
}

func TestHook_Implements(t *testing.T) {
	assert := asserts.NewTestingAsserts(t, true)

	var r packer.Hook
	h := &hook{nil}

	assert.Implementor(h, &r, "should be a Hook")
}

func TestHook_cancelWhileRun(t *testing.T) {
	var finishLock sync.Mutex
	finishOrder := make([]string, 0, 2)

	h := &packer.MockHook{
		RunFunc: func() error {
			time.Sleep(100 * time.Millisecond)

			finishLock.Lock()
			finishOrder = append(finishOrder, "run")
			finishLock.Unlock()
			return nil
		},
	}

	// Serve
	server := rpc.NewServer()
	RegisterHook(server, h)
	address := serveSingleConn(server)

	// Create the client over RPC and run some methods to verify it works
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	hClient := Hook(client)

	// Start the run
	finished := make(chan struct{})
	go func() {
		hClient.Run("foo", nil, nil, nil)
		close(finished)
	}()

	// Cancel it pretty quickly.
	time.Sleep(10 * time.Millisecond)
	hClient.Cancel()

	finishLock.Lock()
	finishOrder = append(finishOrder, "cancel")
	finishLock.Unlock()

	// Verify things are good
	<-finished

	// Check the results
	expected := []string{"cancel", "run"}
	if !reflect.DeepEqual(finishOrder, expected) {
		t.Fatalf("bad: %#v", finishOrder)
	}
}
