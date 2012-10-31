// © 2011 Steve McCoy. Available under the MIT License. See LICENSE for details.

package main

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"net/http"
	"time"
)

type Page struct {
	Id   int64
	Time time.Time
	Body []byte
}

func (p *Page) save(c appengine.Context) (id int64, err error) {
	if len(p.Body) > maxPasteLen {
		err = errors.New("Paste is too large to store")
		return
	}
	k, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Page", nil), p)
	if err != nil {
		return
	}
	return k.IntID(), nil
}

func loadPage(r *http.Request, id int64) (*Page, error) {
	c := appengine.NewContext(r)
	p := new(Page)
	err := datastore.Get(c, datastore.NewKey(c, "Page", "", id, nil), p)
	if err != nil {
		return nil, err
	}
	p.Id = id
	return p, nil
}

func deleteOldPages(c appengine.Context) error {
	q := datastore.NewQuery("Page").
		Filter("Time <", time.Now().Add(-30*24*time.Hour)).
		KeysOnly()

	keys, err := q.GetAll(c, nil)
	if err != nil {
		c.Debugf("Problem with query: %v", err)
		return err
	}

	sz := 500 // datastore delete limit
	n := len(keys) / sz
	for i := 0; i <= n; i++ {
		err = datastore.DeleteMulti(c, window(keys, i*n, i*n+sz))
		if err != nil {
			c.Debugf("Problem with delete: %v", err)
			return err
		}
	}

	return nil
}

func window(keys []*datastore.Key, b, e int) []*datastore.Key {
	if e > len(keys) {
		e = len(keys)
	}
	return keys[b:e]
}
