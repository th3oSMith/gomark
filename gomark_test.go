package gomark_test

import (
	"github.com/th3osmith/gomark"
	"reflect"
	"testing"
)

func TestTag(t *testing.T) {

	b := gomark.NewBookmark()

	b.AddTags("tata", "yoyo", "gateau")

	if len(b.GetTags()) != 3 {
		t.Errorf("Error in tag creation: expected 3 elements got %v", len(b.GetTags()))

	}

	if !b.HasTags("tata", "yoyo", "gateau") {
		t.Errorf("Error in AddTags: got %v expected %v", b.GetTags(), []string{"tata", "yoyo", "gateau"})
	}

	b.DeleteTags("yoyo", "popo")
	if b.HasTags("yoyo") {
		t.Error("Error in Tag Delete")
	}
}

func TestBookmark(t *testing.T) {

	b, err := gomark.NewBookmarkUrl("http://google.com")
	if err != nil {
		t.Errorf("Error while creating bookmark for google.com: %v", err)
	}

	if b.Title != "Google" {
		t.Errorf("Error fetching the title: got %s, expected Google", b.Title)
	}
}

func TestDatabase(t *testing.T) {

	d := gomark.NewDatabase()
	b, _ := gomark.NewBookmarkUrl("http://google.com")

	b.AddTags("TATA", "Yoyo")

	d.AddBookmark(b)

	if len(d.Bookmarks) != 1 {
		t.Errorf("Error while adding bookmark")
	}

	err := d.Dump("/tmp/dump.json")
	if err != nil {
		t.Errorf("Error while dumping to JSON: %s", err)
	}

	d1, err := gomark.NewDatabaseFromFile("/tmp/dump.json")
	if err != nil {
		t.Errorf("Error while reading from JSON: %s", err)
	}

	if !reflect.DeepEqual(d, d1) {
		t.Errorf("Error after loading from JSON, databases not identical: %s, %s", d, d1)
	}

	d.DeleteBookmark(b)

	if len(d.Bookmarks) != 0 {
		t.Errorf("Error while deleting bookmark")
	}

}
