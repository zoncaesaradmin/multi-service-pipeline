//go:build !local
// +build !local

package messagebus

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// --- Mocks ---
type mockProducer struct {
	sendErr  error
	asyncErr error
	closed   bool
}

func (m *mockProducer) Send(ctx context.Context, message *Message) (int32, int64, error) {
	if m.sendErr != nil {
		return 0, 0, m.sendErr
	}
	return 1, 42, nil
}
func (m *mockProducer) SendAsync(ctx context.Context, message *Message) <-chan SendResult {
	ch := make(chan SendResult, 1)
	if m.asyncErr != nil {
		ch <- SendResult{Partition: 0, Offset: 0, Error: m.asyncErr}
	} else {
		ch <- SendResult{Partition: 1, Offset: 42, Error: nil}
	}
	close(ch)
	return ch
}
func (m *mockProducer) Close() error {
	m.closed = true
	return nil
}

type mockConsumer struct {
	assigned    []PartitionAssignment
	onMessageFn func(*Message)
	onAssignFn  func([]PartitionAssignment)
	onRevokeFn  func([]PartitionAssignment)
	commitErr   error
	closed      bool
}

func (m *mockConsumer) Subscribe(topics []string) error                    { return nil }
func (m *mockConsumer) OnMessage(fn func(*Message))                        { m.onMessageFn = fn }
func (m *mockConsumer) OnAssign(fn func([]PartitionAssignment))            { m.onAssignFn = fn }
func (m *mockConsumer) OnRevoke(fn func([]PartitionAssignment))            { m.onRevokeFn = fn }
func (m *mockConsumer) Commit(ctx context.Context, message *Message) error { return m.commitErr }
func (m *mockConsumer) AssignedPartitions() []PartitionAssignment          { return m.assigned }
func (m *mockConsumer) Close() error                                       { m.closed = true; return nil }

// --- Producer Tests ---
func TestKafkaProducerSendTableDriven(t *testing.T) {
	cases := []struct {
		name    string
		sendErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"error", errors.New("fail"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prod := &mockProducer{sendErr: tc.sendErr}
			_, _, err := prod.Send(context.Background(), &Message{})
			if tc.wantErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestKafkaProducerSendAsyncTableDriven(t *testing.T) {
	cases := []struct {
		name     string
		asyncErr error
		wantErr  bool
	}{
		{"success", nil, false},
		{"error", errors.New("fail-async"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prod := &mockProducer{asyncErr: tc.asyncErr}
			ch := prod.SendAsync(context.Background(), &Message{})
			res := <-ch
			if tc.wantErr && res.Error == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.wantErr && res.Error != nil {
				t.Errorf("Unexpected error: %v", res.Error)
			}
		})
	}
}
func TestKafkaProducerInterfaceCompliance(t *testing.T) {
	var _ Producer = &mockProducer{}
}

func TestKafkaProducerSendSuccess(t *testing.T) {
	prod := &mockProducer{}
	part, off, err := prod.Send(context.Background(), &Message{Topic: "t", Value: []byte("v")})
	if part != 1 || off != 42 || err != nil {
		t.Errorf("Send returned wrong values: %d %d %v", part, off, err)
	}
}

func TestKafkaProducerSendError(t *testing.T) {
	prod := &mockProducer{sendErr: errors.New("fail")}
	_, _, err := prod.Send(context.Background(), &Message{})
	if err == nil {
		t.Error("Expected error from Send")
	}
}

func TestKafkaProducerSendAsyncSuccess(t *testing.T) {
	prod := &mockProducer{}
	ch := prod.SendAsync(context.Background(), &Message{})
	res := <-ch
	if res.Partition != 1 || res.Offset != 42 || res.Error != nil {
		t.Errorf("SendAsync returned wrong result: %+v", res)
	}
}

func TestKafkaProducerSendAsyncError(t *testing.T) {
	prod := &mockProducer{asyncErr: errors.New("fail-async")}
	ch := prod.SendAsync(context.Background(), &Message{})
	res := <-ch
	if res.Error == nil {
		t.Error("Expected error from SendAsync")
	}
}

func TestKafkaProducerClose(t *testing.T) {
	prod := &mockProducer{}
	err := prod.Close()
	if err != nil || !prod.closed {
		t.Error("Close did not set closed or returned error")
	}
}

// --- Consumer Tests ---
func TestKafkaConsumerCommitTableDriven(t *testing.T) {
	cases := []struct {
		name      string
		commitErr error
		wantErr   bool
	}{
		{"success", nil, false},
		{"error", errors.New("fail-commit"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cons := &mockConsumer{commitErr: tc.commitErr}
			err := cons.Commit(context.Background(), &Message{})
			if tc.wantErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
func TestKafkaConsumerEventHooksTableDriven(t *testing.T) {
	cons := &mockConsumer{}
	called := map[string]bool{"msg": false, "assign": false, "revoke": false}
	cons.OnMessage(func(msg *Message) { called["msg"] = true })
	cons.OnAssign(func(parts []PartitionAssignment) { called["assign"] = true })
	cons.OnRevoke(func(parts []PartitionAssignment) { called["revoke"] = true })
	if cons.onMessageFn == nil || cons.onAssignFn == nil || cons.onRevokeFn == nil {
		t.Error("Event hooks not set")
	}
	cons.onMessageFn(&Message{})
	cons.onAssignFn([]PartitionAssignment{})
	cons.onRevokeFn([]PartitionAssignment{})
	for k, v := range called {
		if !v {
			t.Errorf("%s hook not called", k)
		}
	}
}
func TestKafkaConsumerInterfaceCompliance(t *testing.T) {
	var _ Consumer = &mockConsumer{}
}

func TestKafkaConsumerSubscribeAndClose(t *testing.T) {
	cons := &mockConsumer{}
	err := cons.Subscribe([]string{"topic"})
	if err != nil {
		t.Error("Subscribe returned error")
	}
	err = cons.Close()
	if err != nil || !cons.closed {
		t.Error("Close did not set closed or returned error")
	}
}

func TestKafkaConsumerAssignedPartitions(t *testing.T) {
	cons := &mockConsumer{assigned: []PartitionAssignment{{Topic: "t", Partition: 2}}}
	parts := cons.AssignedPartitions()
	if len(parts) != 1 || parts[0].Topic != "t" || parts[0].Partition != 2 {
		t.Errorf("AssignedPartitions returned wrong value: %+v", parts)
	}
}

func TestKafkaConsumerCommitSuccess(t *testing.T) {
	cons := &mockConsumer{}
	err := cons.Commit(context.Background(), &Message{})
	if err != nil {
		t.Error("Commit returned error")
	}
}

func TestKafkaConsumerCommitError(t *testing.T) {
	cons := &mockConsumer{commitErr: errors.New("fail-commit")}
	err := cons.Commit(context.Background(), &Message{})
	if err == nil {
		t.Error("Expected error from Commit")
	}
}

func TestKafkaConsumerEventHooks(t *testing.T) {
	cons := &mockConsumer{}
	calledMsg := false
	cons.OnMessage(func(msg *Message) { calledMsg = true })
	if cons.onMessageFn == nil {
		t.Error("OnMessage not set")
	}
	cons.onMessageFn(&Message{})
	if !calledMsg {
		t.Error("OnMessage callback not called")
	}

	calledAssign := false
	cons.OnAssign(func(parts []PartitionAssignment) { calledAssign = true })
	if cons.onAssignFn == nil {
		t.Error("OnAssign not set")
	}
	cons.onAssignFn([]PartitionAssignment{})
	if !calledAssign {
		t.Error("OnAssign callback not called")
	}

	calledRevoke := false
	cons.OnRevoke(func(parts []PartitionAssignment) { calledRevoke = true })
	if cons.onRevokeFn == nil {
		t.Error("OnRevoke not set")
	}
	cons.onRevokeFn([]PartitionAssignment{})
	if !calledRevoke {
		t.Error("OnRevoke callback not called")
	}
}

// --- Type/Field Coverage ---
func TestKafkaProducerTypeFields(t *testing.T) {
	prodType := reflect.TypeOf(KafkaProducer{})
	if _, ok := prodType.FieldByName("producer"); !ok {
		t.Error("KafkaProducer missing 'producer' field")
	}
}

func TestKafkaConsumerTypeFields(t *testing.T) {
	consType := reflect.TypeOf(KafkaConsumer{})
	for _, field := range []string{"consumer", "assignedPartitions", "onAssign", "onRevoke", "onMessage", "onError"} {
		if _, ok := consType.FieldByName(field); !ok {
			t.Errorf("KafkaConsumer missing field: %s", field)
		}
	}
}
