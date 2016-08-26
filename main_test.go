package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/coreos/go-systemd/sdjournal"
)

var (
	EVENT = &sdjournal.JournalEntry{
		Fields:             map[string]string{"_MACHINE_ID": "a8503c78c5a5473f86f44dff20bb348e", "_PID": "1969", "MESSAGE": "{\"foo\":\"bar\"}"},
		Cursor:             "s=1008e329c3074d5fb73aadc2593e1fd4;i=304;b=933ab8c8a0f84defa0d3bc16578bd30f;m=1104d38;t=538cc987de325;x=337a1230e7215dbc",
		RealtimeTimestamp:  0x538cc987de325,
		MonotonicTimestamp: 0x1104d38}
)

//func TestFoo(t *testing.T) {
//
//	c := jbconf{}
//	b, err := json.Marshal(c)
//	if err != nil {
//		t.Error("unexpected error", err)
//	}
//	fmt.Println(string(b))
//
//}

func tempfile(b []byte) (string, error) {
	f, err := ioutil.TempFile("", "config-")
	if err != nil {
		return "", err
	}
	f.Write(b)
	f.Close()
	return f.Name(), nil
}

func TestConfigure(t *testing.T) {

	f, _ := tempfile([]byte(`{"elasticsearch_url":"http://localhost:9200"}`))
	defer os.Remove(f)
	_, err := configure(f)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	f, _ = tempfile([]byte(`{"elasticsearch_url":1}`))
	defer os.Remove(f)
	_, err = configure(f)
	if !strings.HasSuffix(fmt.Sprint(err), "cannot unmarshal number into Go value of type string") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCommit(t *testing.T) {

	f := "foo"
	err := commit(f, "bar")
	defer os.Remove(f)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	b, _ := ioutil.ReadFile(f)
	if string(b) != "bar\n" {
		t.Errorf("expected %q got %q", "bar", string(b))
	}

	f = "/nosuchdir/nosuchfile"
	err = commit(f, "baz")
	if err == nil {
		t.Error("expected error")
	}
	defer os.Remove(f)

}

func TestFormat(t *testing.T) {

	c := &jbconf{}
	m := format(c, EVENT)
	j := m.Source["journal"].(map[string]interface{})

	if _, ok := j["_MACHINE_ID"].(string); !ok {
		t.Errorf("expected string, got: %T", m.Source["_MACHINE_ID"])
	}
	if _, ok := j["_PID"].(int64); !ok {
		t.Errorf("expected int, got: %T", m.Source["_PID"])
	}
	if m.Source["@timestamp"] != "2016-07-29T21:04:26.424Z" {
		t.Errorf("expected %q got %q", "2016-07-29T21:04:26.424Z", m.Source["@timestamp"])
	}
	if m.Source["message"] != `{"foo":"bar"}` {
		t.Error("message not as expected")
	}
	// by default, do not parse json
	if _, ok := m.Source["structured"]; ok {
		t.Error("expected nil")
	}

	c = &jbconf{ParseJson: true}
	m = format(c, EVENT)
	if m.Source["structured"].(map[string]interface{})["foo"] != "bar" {
		t.Error("foo != bar")
	}
}

func BenchmarkFormat(b *testing.B) {

	// on my laptop in vm this records 5k ns/op
	c := &jbconf{}
	for i := 0; i < b.N; i++ {
		format(c, EVENT)
	}

}

func TestTail(t *testing.T) {

	_, err := tail("nosuchcursor")
	if err == nil {
		t.Error("expected error")
	}

	j, _ := tail("")
	e := <-j
	fmt.Println(e)

}
