package waha

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(srv *httptest.Server) *Client {
	return New(Config{BaseURL: srv.URL, APIKey: "k", Session: "default"})
}

func TestGetSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "k" {
			t.Errorf("missing api key header")
		}
		if r.URL.Path != "/api/sessions/default" {
			t.Errorf("path = %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"name": "default", "status": "WORKING",
			"me": map[string]string{"id": "628111@c.us", "pushName": "Biz"},
		})
	}))
	defer srv.Close()
	s, err := testClient(srv).GetSession()
	if err != nil || s.Status != "WORKING" || s.Me.PushName != "Biz" {
		t.Fatalf("got %+v err %v", s, err)
	}
}

func TestSendTextNormalizesID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sendText" {
			t.Errorf("path = %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req SendTextReq
		json.Unmarshal(body, &req)
		if req.ChatID != "628999@c.us" || req.Session != "default" {
			t.Errorf("req = %+v", req)
		}
		// WEBJS shape: nested id._serialized
		json.NewEncoder(w).Encode(map[string]any{"id": map[string]string{"_serialized": "MSG_1"}})
	}))
	defer srv.Close()
	res, err := testClient(srv).SendText("628999@c.us", "hi")
	if err != nil || res.MessageID() != "MSG_1" {
		t.Fatalf("got %+v err %v", res, err)
	}
}

func TestSendTextNowebID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"key": map[string]string{"id": "K_2"}})
	}))
	defer srv.Close()
	res, _ := testClient(srv).SendText("628@c.us", "hi")
	if res.MessageID() != "K_2" {
		t.Fatalf("noweb id = %q", res.MessageID())
	}
}

func TestPhoneToChatID(t *testing.T) {
	if got := PhoneToChatID("0812-345"); got != "62812345@c.us" {
		t.Fatalf("chatID = %q", got)
	}
}

func TestErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := testClient(srv).GetSession(); err == nil {
		t.Fatal("expected error on 500")
	}
}
