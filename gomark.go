package gomark

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type database struct {
	Revision  int
	Bookmarks map[string]bookmark
}

func (d *database) AddBookmark(b *bookmark) {
	d.Bookmarks[b.GetURL()] = *b
}

func (d *database) DeleteBookmark(b *bookmark) {
	delete(d.Bookmarks, b.GetURL())
}

func (d *database) Dump(filename string) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, b, 0600)
	return err
}

func NewDatabaseFromFile(filename string) (d *database, err error) {

	d = NewDatabase()

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, d)
	if err != nil {
		return nil, err
	}

	return d, err
}

func NewDatabase() (d *database) {
	d = new(database)
	d.Bookmarks = make(map[string]bookmark)
	return
}

type bookmark struct {
	Title string
	Date  time.Time
	info  bookmarkInfo // Needed to serialize easily the private attributes
}

type bookmarkInfo struct {
	Url     url.URL
	Tags    map[string]struct{}
	Deleted bool
	ToRead  bool
}

// Custom JSON
// From http://choly.ca/post/go-json-marshalling/

func (d bookmark) MarshalJSON() ([]byte, error) {

	// We need the alias otherwise it would inherite the methods
	// and MarshalJSON would be call infinitely
	type alias bookmark

	return json.Marshal(&struct {
		Info bookmarkInfo
		alias
	}{
		d.info,
		(alias)(d)})
}

func (d *bookmark) UnmarshalJSON(data []byte) error {

	type alias bookmark

	aux := &struct {
		Info bookmarkInfo
		*alias
	}{
		d.info,
		(*alias)(d)}

	err := json.Unmarshal(data, &aux)

	if err != nil {
		return err
	}

	d.info = aux.Info

	return nil
}

func NewBookmark() *bookmark {

	b := new(bookmark)
	b.info.Tags = make(map[string]struct{})
	b.Date = time.Now()

	return b
}

func NewBookmarkUrl(rawUrl string) (*bookmark, error) {

	tmp, err := url.Parse(rawUrl)

	if err != nil {
		return nil, err
	}

	b := NewBookmark()
	b.info.Url = *tmp

	res, err := http.Get(rawUrl)

	if err != nil {
		// As parsing the title is non mandatory no error is returned
		log.Printf("Error while getting the page %s: %s", rawUrl, err.Error())
		return b, nil
	}

	// Getting the title
	head := make([]byte, 2000)
	_, err = res.Body.Read(head)

	if err != nil {
		// As parsing the title is non mandatory no error is returned
		log.Printf("Error while reading the page %s: %s", rawUrl, err.Error())
		return b, nil
	}

	re := regexp.MustCompile("<title>(.+)</title>")
	matches := re.FindStringSubmatch(string(head))

	b.Title = matches[1]

	// TODO Fallback to the entire file if pb

	return b, nil

}

func (b *bookmark) AddTags(tags ...string) {

	for _, tag := range tags {
		tag = strings.ToLower(tag)
		if _, found := b.info.Tags[tag]; !found {
			b.info.Tags[tag] = struct{}{}
		}
	}
}

func (b *bookmark) DeleteTags(tags ...string) {
	for _, tag := range tags {
		tag = strings.ToLower(tag)
		delete(b.info.Tags, tag)
	}
}

func (b *bookmark) GetTags(tags ...string) (tagSet []string) {

	for tag := range b.info.Tags {
		tagSet = append(tagSet, tag)
	}

	return
}

func (b *bookmark) HasTags(tags ...string) bool {
	for _, tag := range tags {
		tag = strings.ToLower(tag)
		if _, found := b.info.Tags[tag]; !found {
			return false
		}
	}

	return true
}

func (b *bookmark) Remove() {
	b.info.Deleted = true
}

func (b *bookmark) UnRemove() {
	b.info.Deleted = false
}

func (b *bookmark) Read() {
	b.info.ToRead = false
}

func (b *bookmark) UnRead() {
	b.info.ToRead = true
}

func (b *bookmark) GetURL() string {
	return b.info.Url.String()
}
