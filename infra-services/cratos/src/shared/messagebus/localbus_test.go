//go:build local
// +build local

package messagebus

import (
	"context"
	"testing"
	"time"
)

const sendFailedMsg = "Send failed: %v"

func TestLocalProducerImplementsProducer(t *testing.T) {
	var _ Producer = &LocalProducer{}
}

func TestLocalConsumerImplementsConsumer(t *testing.T) {
	var _ Consumer = &LocalConsumer{}
}

func TestLocalProducerSendAndSendAsync(t *testing.T) {
	CleanupMessageBus()
	prod := &LocalProducer{}
	msg := &Message{Topic: "testtopic", Value: []byte("hello")}
	part, off, err := prod.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf(sendFailedMsg, err)
	}
	if part != 0 || off != 0 {
		t.Errorf("Send returned wrong partition/offset: %d/%d", part, off)
	}
	ch := prod.SendAsync(context.Background(), &Message{Topic: "testtopic", Value: []byte("async")})
	res := <-ch
	if res.Error != nil {
		t.Errorf("SendAsync returned error: %v", res.Error)
	}
}

func TestLocalProducerClose(t *testing.T) {
	prod := &LocalProducer{}
	if err := prod.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestLocalConsumerSubscribeAndPoll(t *testing.T) {
	CleanupMessageBus()
	prod := &LocalProducer{}
	cons := NewConsumer(map[string]any{}, "testcgroup").(*LocalConsumer)
	received := make(chan *Message, 1)
	cons.OnMessage(func(msg *Message) {
		received <- msg
	})
	if err := cons.Subscribe([]string{"polltopic"}); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	msg := &Message{Topic: "polltopic", Value: []byte("data")}
	_, _, err := prod.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf(sendFailedMsg, err)
	}
	select {
	case got := <-received:
		if got == nil || string(got.Value) != "data" {
			t.Errorf("OnMessage did not return expected message")
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("OnMessage callback not triggered in time")
	}
	cons.Close()
}

func TestLocalConsumerCommitAndClose(t *testing.T) {
	cons := &LocalConsumer{lastRead: make(map[string]int64)}
	msg := &Message{Topic: "commit", Value: []byte("data")}
	if err := cons.Commit(context.Background(), msg); err != nil {
		t.Errorf("Commit returned error: %v", err)
	}
	if err := cons.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestCleanupMessageBusAndStats(t *testing.T) {
	CleanupMessageBus()
	prod := &LocalProducer{}
	msg := &Message{Topic: "statstopic", Value: []byte("stat")}
	_, _, err := prod.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf(sendFailedMsg, err)
	}
	stats := GetMessageBusStats()
	if stats["status"] == "no_messages" {
		t.Errorf("Stats should show messages present")
	}
	if err := CleanupMessageBus(); err != nil {
		t.Errorf("CleanupMessageBus returned error: %v", err)
	}
	stats = GetMessageBusStats()
	if stats["status"] != "no_messages" {
		t.Errorf("Stats should show no messages after cleanup")
	}
}

func TestLocalConsumerInterfaceHooks(t *testing.T) {
	cons := &LocalConsumer{lastRead: make(map[string]int64), watcherDone: make(chan struct{})}
	assignCalled := false
	revokeCalled := false
	msgCalled := false
	cons.OnAssign(func(parts []PartitionAssignment) { assignCalled = true })
	cons.OnRevoke(func(parts []PartitionAssignment) { revokeCalled = true })
	cons.OnMessage(func(msg *Message) { msgCalled = true })
	cons.Subscribe([]string{"hooktopic"})
	if !assignCalled {
		t.Error("OnAssign not called")
	}
	cons.Subscribe([]string{}) // triggers revoke
	if !revokeCalled {
		t.Error("OnRevoke not called")
	}
	cons.onMessage(&Message{})
	if !msgCalled {
		t.Error("OnMessage not called")
	}
}
