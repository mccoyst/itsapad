package main

import (
	"appengine"
	"appengine/datastore"
	"os"
	"http"
)

type Page struct {
	Id   int64
	Time datastore.Time
	Body []byte
}

func (p *Page) save(c appengine.Context) (id int64, err os.Error) {
	if len(p.Body) > maxPasteLen {
		err = os.NewError("Paste is too large to store")
		return
	}
	k, err := datastore.Put(c, datastore.NewIncompleteKey("Page"), p)
	if err != nil {
		return
	}
	return k.IntID(), nil
}

func loadPage(r *http.Request, id int64) (*Page, os.Error) {
	c := appengine.NewContext(r)
	p := new(Page)
	err := datastore.Get(c, datastore.NewKey("Page", "", id, nil), p)
	if err != nil {
		return nil, err
	}
	p.Id = id
	return p, nil
}
