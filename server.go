package gomark

import (
	"encoding/json"
	"fmt"
	"github.com/th3osmith/pure"
	"log"
	"net/http"
)

type bookmarkHandler struct {
	database *Database
}

type BookmarkJSON struct {
	Url  string
	Tags []string
}

func (h bookmarkHandler) Create(m pure.PureReq, rw pure.ResponseWriter) {

	rww := rw.(*pure.PureResponseWriter)
	msg := m.Msg

	data := BookmarkJSON(msg.RequestMap["data"].(BookmarkJSON))

	b, err := NewBookmarkUrl(data.Url)
	if err != nil {
		rww.AddLogMsg(pure.Error, 500, "Impossible to create bookmark")
		rww.Fail()
		return
	}

	if len(data.Tags) > 0 {
		b.AddTags(data.Tags...)
	}

	h.database.AddBookmark(b)

	result := make(map[string]Bookmark)
	result[data.Url] = *b
	rww.AddValue("result", result)

	rww.AddLogMsg(pure.Info, 200, fmt.Sprintf("Created Bookmark for %s", data.Url))
	h.database.Dump()

	return
}

func (h bookmarkHandler) Update(m pure.PureReq, rw pure.ResponseWriter) {

	rww := rw.(*pure.PureResponseWriter)
	msg := m.Msg

	data, dataOk := msg.RequestMap["data"].(BookmarkJSON)
	add_tmp, add_ok := msg.RequestMap["add_tags"]
	del_tmp, del_ok := msg.RequestMap["del_tags"]
	url := msg.RequestMap["url"].(string)

	b, err := h.database.GetBookmark(url)
	if err != nil {
		rww.AddLogMsg(pure.Error, 500, "Impossible to get bookmark")
		rww.AddLogMsg(pure.Error, 500, err.Error())
		rww.Fail()
		return
	}

	if dataOk && len(data.Tags) > 0 {
		b.ResetTags()
		b.AddTags(data.Tags...)
	}

	if add_ok {
		b.AddTags(add_tmp.([]string)...)
	}

	if del_ok {
		b.DeleteTags(del_tmp.([]string)...)
	}

	h.database.AddBookmark(b)

	result := make(map[string]Bookmark)
	result[url] = *b
	rww.AddValue("result", result)

	rww.AddLogMsg(pure.Info, 200, fmt.Sprintf("Updated Bookmark for %s", data.Url))
	h.database.Dump()

	return
}

func (h bookmarkHandler) Delete(m pure.PureReq, rw pure.ResponseWriter) {

	rww := rw.(*pure.PureResponseWriter)
	msg := m.Msg

	url := msg.RequestMap["url"].(string)

	b, err := h.database.GetBookmark(url)
	if err != nil {
		rww.AddLogMsg(pure.Error, 500, "Impossible to get bookmark")
		rww.AddLogMsg(pure.Error, 500, err.Error())
		rww.Fail()
		return
	}

	h.database.DeleteBookmark(b)

	result := make(map[string]Bookmark)
	result[url] = *b
	rww.AddValue("result", result)

	rww.AddLogMsg(pure.Info, 200, fmt.Sprintf("Deleted Bookmark for %s", url))
	h.database.Dump()

	return

}
func (h bookmarkHandler) Retrieve(m pure.PureReq, rw pure.ResponseWriter) {

	rww := rw.(*pure.PureResponseWriter)
	msg := m.Msg

	url := msg.RequestMap["url"].(string)

	result := make(map[string]Bookmark)

	if len(url) == 0 {
		result = h.database.GetBookmarks()
		rww.AddLogMsg(pure.Info, 200, fmt.Sprintf("Retrieved all Bookmarks"))

	} else {
		b, err := h.database.GetBookmark(url)
		if err != nil {
			rww.AddLogMsg(pure.Error, 500, "Impossible to get bookmark")
			rww.AddLogMsg(pure.Error, 500, err.Error())
			rww.Fail()
			return
		}
		result[url] = *b
		rww.AddLogMsg(pure.Info, 200, fmt.Sprintf("Retrieved Bookmark for %s", url))

	}

	rww.AddValue("result", result)

}

func (h bookmarkHandler) Flush(m pure.PureReq, rw pure.ResponseWriter) {
}

type Server struct {
	Muxer   *pure.PureMux
	Handler *bookmarkHandler
}

type RequestMap struct {
	Url     string       `json:"url"`
	Data    BookmarkJSON `json:"data"`
	AddTags []string     `json:"add_tags"`
	DelTags []string     `json:"del_tags"`
}

func DecodeRequestMap(p json.RawMessage) (err error, out map[string]interface{}) {

	out = make(map[string]interface{})

	var rm RequestMap
	err = json.Unmarshal(p, &rm)
	if err != nil {
		return
	}

	out["add_tags"] = rm.AddTags
	out["del_tags"] = rm.DelTags
	out["url"] = rm.Url
	out["data"] = rm.Data

	return
}

type authMiddleware struct {
	authenticator Authenticator
}

func (am authMiddleware) Auth(req pure.PureReq, rw pure.ResponseWriter) bool {

	rww := rw.(*pure.PureResponseWriter)

	username, okUsername := req.Msg.TransactionMap["username"]
	password, okPassword := req.Msg.TransactionMap["password"]

	if okUsername && okPassword &&
		am.authenticator.CheckCredentials(username, password) {
		return true
	}

	rww.AddLogMsg(pure.Error, 403, "Access Denied")
	return false

}

type HttpConfig struct {
	UseTLS          bool
	CertificateFile string
	KeyFile         string
	Authenticator   Authenticator
}

type Authenticator interface {
	CheckCredentials(username string, password string) bool
}

func ServeHttp(db *Database, server *Server, port int, config HttpConfig) {

	Serve(db, server, config.Authenticator)

	http.Handle("/pure", pure.WebsocketHandler(*server.Muxer, DecodeRequestMap))

	var err error

	if config.UseTLS {
		err = http.ListenAndServeTLS(fmt.Sprintf(":%v", port), config.CertificateFile, config.KeyFile, nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	}

	log.Fatal(err)

}

func Serve(db *Database, server *Server, authenticator Authenticator) {

	mux := pure.NewPureMux()

	h := bookmarkHandler{db}

	if authenticator != nil {
		am := authMiddleware{authenticator}
		hb := pure.AddMiddleware(h, am.Auth)
		mux.RegisterHandler("bookmark", hb)

	} else {
		mux.RegisterHandler("bookmark", h)
	}

	server.Muxer = mux
	server.Handler = &h

}
