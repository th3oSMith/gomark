package gomark_test

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/th3osmith/gomark"
	"github.com/th3osmith/pure"
	"net/url"
	"strings"
	"testing"
)

func TestWebsocketServer(t *testing.T) {

	db := gomark.NewDatabase()

	var server gomark.Server
	go gomark.ServeHttp(db, &server, 3000, gomark.HttpConfig{})

	u := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/pure"}

	dialer := websocket.Dialer{}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		t.Error("dial:", err)
		return
	}
	defer c.Close()

	confirmation := make(chan bool)

	go func() {
		defer c.Close()
		first := true
		for {
			_, p, err := c.ReadMessage()
			if err != nil {
				t.Error("Error Reading websocket:", err)
				confirmation <- false
				return
			}

			if first {
				first = false
				continue
			}
			if !strings.Contains(string(p), "CREATED") {
				c.Close()
				t.Error("Error using websocket pure", string(p))
				return
			}
			confirmation <- true
			return

		}

	}()

	mm := make(map[string]interface{})
	data := gomark.BookmarkJSON{"http://google.com", []string{"tata", "yoyo"}}
	mm["data"] = data

	// Creating Request
	req := pure.PureMsg{DataType: "bookmark", Action: "create", RequestMap: mm}
	jsonReq, _ := json.Marshal(req)

	err = c.WriteMessage(websocket.TextMessage, jsonReq)
	<-confirmation

}

func TestServer(t *testing.T) {

	db := gomark.NewDatabase()

	var server gomark.Server
	gomark.Serve(db, &server, nil)

	c1 := pure.GoConn{Response: make(chan pure.PureMsg, 1), Muxer: server.Muxer}

	mm := make(map[string]interface{})

	data := gomark.BookmarkJSON{"http://google.com", []string{"tata", "yoyo"}}
	mm["data"] = data

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "create", RequestMap: mm})
	resp := c1.ReadResp()

	if resp.Action != "CREATED" {
		t.Errorf("Error in the creation of the bookmark: %s", resp)
	}

	mn := make(map[string]interface{})
	mn["url"] = "http://google.com"
	mn["add_tags"] = []string{"yaourt", "pomme"}

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "update", RequestMap: mn})
	resp = c1.ReadResp()

	if resp.Action != "UPDATED" {
		t.Error("Error in Update (add tags)", resp)
	}

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "retrieve", RequestMap: mn})
	resp = c1.ReadResp()

	if resp.Action != "RETRIEVED" {
		t.Error("Error in Retrieve")
	}

	books := resp.ResponseMap["result"].(map[string]gomark.Bookmark)
	book := books["http://google.com"]

	if len(book.GetTags()) != 4 {
		t.Errorf("Error in update: expected 4 tags got %s", len(book.GetTags()))
	}

	mn["del_tags"] = []string{"yaourt", "pomme"}
	delete(mn, "add_tags")

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "update", RequestMap: mn})
	resp = c1.ReadResp()

	if resp.Action != "UPDATED" {
		t.Error("Error in Update (del tags)")
	}

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "retrieve", RequestMap: mn})
	resp = c1.ReadResp()

	if resp.Action != "RETRIEVED" {
		t.Error("Error in Retrieve")
	}

	books = resp.ResponseMap["result"].(map[string]gomark.Bookmark)
	book = books["http://google.com"]

	if len(book.GetTags()) != 2 {
		t.Errorf("Error in update: expected 2 tags got %s", len(book.GetTags()))
	}

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "retrieve", RequestMap: mn})
	resp = c1.ReadResp()

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "update", RequestMap: mn})
	resp = c1.ReadResp()

	mn["url"] = "http://google.coma"

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "retrieve", RequestMap: mn})
	resp = c1.ReadResp()

	if resp.Action != "RETRIEVE_FAIL" {
		t.Errorf("Error in Retrieve not existent: %s", resp)
	}

	data = gomark.BookmarkJSON{"http://google.fr", []string{"tata", "yoyo"}}
	mm["data"] = data

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "create", RequestMap: mm})
	c1.ReadResp()

	mm = make(map[string]interface{})
	mm["url"] = ""

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "retrieve", RequestMap: mm})
	resp = c1.ReadResp()

	if resp.Action != "RETRIEVED" {
		t.Error("Error in Retrieve multiple bookmarks")
	}

	mn["url"] = "http://google.com"

	c1.SendReq(pure.PureMsg{DataType: "bookmark", Action: "delete", RequestMap: mn})
	resp = c1.ReadResp()

	if resp.Action != "DELETED" {
		t.Errorf("Error in Delete: %s", resp)
	}

}
