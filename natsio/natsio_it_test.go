package natsio

import (
	"github.com/byrnedo/prefab"
	. "github.com/byrnedo/apibase/natsio/protobuf"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/encoders/protobuf"
	"os"
	"reflect"
	"testing"
	"time"
)


var (
	natsContainer string
	natsUrl string
)

type Wrap struct {
	*TestMessage
}

func (w *Wrap) SetContext(ctx *NatsContext) {
	w.Context = ctx
}

func WrapMessage(msg *TestMessage) *Wrap {
	return &Wrap{msg}
}

func setup() {
	nats.RegisterEncoder(protobuf.PROTOBUF_ENCODER, &protobuf.ProtobufEncoder{})

	natsContainer, natsUrl = prefab.StartNatsContainer()

}

func TestMain(m *testing.M) {
	// your func
	setup()

	retCode := m.Run()

	// call with result of m.Run()
	prefab.Remove(natsContainer)
	os.Exit(retCode)
}

func TestNewNatsConnect(t *testing.T) {
	natsOpts := setupConnection()

	var natsCon *Nats

	time.Sleep(500 * time.Millisecond)

	natsCon, err := natsOpts.Connect()
	if err != nil {
		t.Error("Failed to connect:" + err.Error())
		return
	}

	var handler = func(subj string, reply string, testData *TestMessage) {
		t.Logf("Got message on nats: %+v", testData)
		//EncCon is nil at this point but that's ok
		//since it wont get called until after connecting
		//when it will then get a ping message.
		data := "Pong"
		natsCon.Publish(reply, WrapMessage(&TestMessage{
			Context: testData.Context,
			Data: &data,
		}))
	}

	var simpleHandler = func(m *nats.Msg) {

		testData := &TestMessage{}
		testData.Unmarshal(m.Data)
		handler(m.Subject, m.Reply, testData)
	}

	natsCon.Subscribe("test.a", handler)

	natsCon.Subscribe("test.b.>", simpleHandler)

	data := "Ping"
	response := TestMessage{}
	request := &TestMessage{
		Context: &NatsContext{},
		Data:    &data,
	}
	err = natsCon.Request("test.a", WrapMessage(request), &response, 2*time.Second)
	t.Logf("Got response on nats: %+v", &response)

	if err != nil {
		t.Error("Failed to get response:" + err.Error())
		return
	}

	if len(response.Context.Trail) != 2 {
		t.Errorf("App trail len is %d, expected 2\n", len(response.Context.Trail))
		return
	}

	if *response.Context.TraceId != *request.Context.TraceId {
		t.Errorf("Request and response trace id differs\nExpected %s\nReceived %s\n", *request.Context.TraceId, *response.Context.TraceId)
	}

	err = natsCon.Request("test.b.some", WrapMessage(request), &response, 2*time.Second)
	t.Logf("Got response from simple handler on nats: %+v", &response)

	if err != nil {
		t.Error("Failed to get response via simple subscribe:" + err.Error())
		return
	}

	natsCon.UnsubscribeAll()
	err = natsCon.EncCon.Request("test.a", request, &response, 2*time.Second)
	if err == nil {
		t.Error("Should have failed to get response")
		return
	}
}

func TestHandleFunc(t *testing.T) {

	var handleFunc1 = func(n *nats.Msg) {}
	var handleFunc2 = func(n *nats.Msg) {}

	natsOpts := setupConnection()

	var natsCon *Nats

	time.Sleep(500 * time.Millisecond)

	natsCon, err := natsOpts.Connect()
	if err != nil {
		t.Error("Failed to connect:" + err.Error())
		return
	}

	natsCon.Subscribe("test.handle_func", handleFunc1)
	natsCon.Subscribe("test.handle_func_2", handleFunc2)

	natsCon.QueueSubscribe("test.handle_func", "group", handleFunc1)
	natsCon.QueueSubscribe("test.handle_func_2", "group_2", handleFunc2)

	expectedRoutes := []*Route{
		&Route{
			"test.handle_func",
			handleFunc1,
			nil,
			"",
		},
		&Route{
			"test.handle_func_2",
			handleFunc2,
			nil,
			"",
		},
		&Route{
			"test.handle_func",
			handleFunc1,
			nil,
			"group",
		},
		&Route{
			"test.handle_func_2",
			handleFunc2,
			nil,
			"group_2",
		},
	}

	routes := natsOpts.GetRoutes()
	if len(routes) != 4 {
		t.Error("Not 4 routes created")
	}

	for ind, route := range routes {
		if route.GetRoute() != expectedRoutes[ind].GetRoute() {
			t.Errorf("Routes not as expected:\nexpected %+v\nactual %+v", expectedRoutes[ind].GetRoute(), route.GetRoute())
		}
		if route.GetSubscription() == nil {
			t.Errorf("Subscr not as expected:\nexpected %+v\nactual %+v", expectedRoutes[ind].GetSubscription(), route.GetSubscription())
		}

		f1 := reflect.ValueOf(route.GetHandler())
		f2 := reflect.ValueOf(expectedRoutes[ind].GetHandler())

		if f1.Pointer() != f2.Pointer() {
			t.Errorf("Handlers not as expected:\nexpected %+v\nactual %+v", f2, f1)
		}
	}

}

func setupConnection() *NatsOptions {

	return NewNatsOptions(func(n *NatsOptions) error {
		n.Url = natsUrl
		n.Name = "it_test"
		n.Timeout = 10 * time.Second
		return nil
	})

}
