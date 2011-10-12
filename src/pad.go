// Copyright © 2011 Steve McCoy
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"appengine"
	"appengine/datastore"
	"http"
	"regexp"
	"strconv"
	"template"
	"time"
)

var maxPasteLen = 32768
var templates = make(map[string]*template.Template)
var viewValidator = regexp.MustCompile("^/([0-9]+)(/([a-z]+)?)?$")

func init() {
	for _, tmpl := range []string{"paste", "plain", "fancy"} {
		t := "tmplt/" + tmpl + ".html"
		templates[tmpl] = template.Must(template.ParseFile(t))
	}

	http.HandleFunc("/", pasteHandler)
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	c.Debugf("path = %s", r.URL.Path)

	if r.Method == "POST" && r.URL.Path == "/" {
		c.Debugf("posting")
		saveHandler(w, r)
		return
	}

	if r.Method == "GET" && r.URL.Path == "/" {
		renderTemplate(w, "paste", new(Page))
		return
	}

	parts := viewValidator.FindStringSubmatch(r.URL.Path)
	if parts == nil {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.Atoi64(parts[1])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	view := parts[3]

	if r.Method == "POST" && view == "" {
		p, err := loadPage(r, id)
		if err != nil {
			p = new(Page)
		} // Oh well
		renderTemplate(w, "paste", p)
		return
	}

	if r.Method == "GET" {
		if view == "" {
			view = "plain"
		}
		p, err := loadPage(r, id)
		if err != nil || view != "plain" && view != "fancy" {
			http.NotFound(w, r)
			return
		}
		renderTemplate(w, view, p)
		return
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	body := r.FormValue("body")
	p := &Page{
		Time: datastore.SecondsToTime(time.Seconds()),
		Body: body,
	}
	id, err := p.save(c)
	if err != nil {
		c.Errorf("Error saving paste %d\n", id)
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	c.Debugf("Saving paste %v\n", id)
	http.Redirect(w, r, strconv.Itoa64(id), http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates[tmpl].Execute(w, p)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
}
