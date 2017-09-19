package gomark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Database struct {
	Bookmarks map[string]Bookmark
	Filename  string
}

func (d *Database) AddBookmark(b *Bookmark) {
	d.Bookmarks[b.GetURL()] = *b
}

func (d *Database) GetBookmarks() map[string]Bookmark {
	return d.Bookmarks
}

func (d *Database) GetBookmark(url string) (b *Bookmark, err error) {

	book, ok := d.Bookmarks[url]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Bookmark not found: %s", url))
	}

	b = &book
	return

}

func (d *Database) DeleteBookmark(b *Bookmark) {
	delete(d.Bookmarks, b.GetURL())
}

func (d *Database) Dump() error {

	if len(d.Filename) == 0 {
		return fmt.Errorf("No file specified")
	}

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(d.Filename, b, 0600)
	return err
}

func NewDatabaseFromFile(filename string) (d *Database, err error) {

	d = NewDatabase()
	d.Filename = filename

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// If the file is empty just return a new DB
	if len(b) == 0 {
		return NewDatabase(), nil
	}

	err = json.Unmarshal(b, d)
	if err != nil {
		return nil, err
	}

	return d, err
}

func NewDatabase() (d *Database) {
	d = new(Database)
	d.Bookmarks = make(map[string]Bookmark)
	return
}

type Bookmark struct {
	Title  string
	Date   time.Time
	RawUrl string
	info   bookmarkInfo // Needed to serialize easily the private attributes
}

type bookmarkInfo struct {
	Url  url.URL
	Tags map[string]struct{}
}

// Custom JSON
// From http://choly.ca/post/go-json-marshalling/

func (d Bookmark) MarshalJSON() ([]byte, error) {

	// We need the alias otherwise it would inherite the methods
	// and MarshalJSON would be call infinitely
	type alias Bookmark

	fmt.Println(d.GetTags())

	return json.Marshal(&struct {
		Url  string
		Tags []string
		alias
	}{
		d.GetURL(),
		d.GetTags(),
		(alias)(d)})
}

func (d *Bookmark) UnmarshalJSON(data []byte) error {

	type alias Bookmark

	aux := &struct {
		Url  string
		Tags []string
		*alias
	}{
		"",
		[]string{},
		(*alias)(d)}

	err := json.Unmarshal(data, &aux)

	if err != nil {
		return err
	}

	url, err := url.Parse(aux.Url)
	if err != nil {
		return err
	}

	d.info.Url = *url
	d.ResetTags()
	d.AddTags(aux.Tags...)

	return nil
}

func NewBookmark() *Bookmark {

	b := new(Bookmark)
	b.info.Tags = make(map[string]struct{})
	b.Date = time.Now()

	return b
}

func NewBookmarkUrl(rawUrl string) (*Bookmark, error) {

	tmp, err := url.Parse(rawUrl)

	if err != nil {
		return nil, err
	}

	b := NewBookmark()
	b.info.Url = *tmp
	b.RawUrl = rawUrl

	res, err := http.Get(rawUrl)

	if err != nil {
		// As parsing the title is non mandatory no error is returned
		log.Printf("Error while getting the page %s: %s", rawUrl, err.Error())
		return b, nil
	}

	// Getting the title
	head := make([]byte, 2000)
	_, err = res.Body.Read(head)

	if err != nil && err != io.EOF {
		// As parsing the title is non mandatory no error is returned
		log.Printf("Error while reading the page %s: %s", rawUrl, err.Error())
		return b, nil
	}

	re := regexp.MustCompile("(?s)<title.*?>(.+)</title>")
	matches := re.FindStringSubmatch(string(head))

	if len(matches) == 0 {
		rest, err := ioutil.ReadAll(res.Body)

		if err != nil && err != io.EOF {
			// As parsing the title is non mandatory no error is returned
			log.Printf("Error while reading the full page %s: %s", rawUrl, err.Error())
			return b, nil
		}

		bodyText := append(head, rest...)
		matches = re.FindStringSubmatch(string(bodyText))
	}

	if len(matches) == 0 {
		b.Title = b.RawUrl
		return b, nil
	}
	b.Title = matches[1]

	return b, nil

}

func (b *Bookmark) ResetTags(tags ...string) {
	b.info.Tags = make(map[string]struct{})
}

func (b *Bookmark) AddTags(tags ...string) {

	for _, tag := range tags {
		tag = strings.ToLower(tag)
		if _, found := b.info.Tags[tag]; !found {
			b.info.Tags[tag] = struct{}{}
		}
	}
}

func (b *Bookmark) DeleteTags(tags ...string) {
	for _, tag := range tags {
		tag = strings.ToLower(tag)
		delete(b.info.Tags, tag)
	}
}

func (b *Bookmark) GetTags(tags ...string) (tagSet []string) {

	for tag := range b.info.Tags {
		tagSet = append(tagSet, tag)
	}

	return
}

func (b *Bookmark) HasTags(tags ...string) bool {
	for _, tag := range tags {
		tag = strings.ToLower(tag)
		if _, found := b.info.Tags[tag]; !found {
			return false
		}
	}

	return true
}

func (b *Bookmark) GetURL() string {
	return b.info.Url.String()
}
