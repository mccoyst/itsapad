// © 2011 Steve McCoy. Available under the MIT License. See LICENSE for details.

package main

import (
	"appengine"
	"net/http"
	"regexp"
	"strconv"
	"text/template"
	"time"
)

var maxPasteLen = 1048576
var templates = make(map[string]*template.Template)
var viewValidator = regexp.MustCompile("^/([0-9]+)(/([a-z]+)?)?$")

func init() {
	for _, tmpl := range []string{"paste", "plain", "fancy"} {
		t := "tmplt/" + tmpl + ".html"
		templates[tmpl] = template.Must(template.ParseFiles(t))
	}

	http.HandleFunc("/", pasteHandler)
	http.HandleFunc("/clean", cleaner)
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

	id, err := strconv.ParseInt(parts[1], 10, 64)
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
		if err != nil || view != "plain" && view != "fancy" && view != "raw" {
			http.NotFound(w, r)
			return
		}

		if view == "raw" {
			w.Header().Set("Content-Type", "text/plain")
			_, err := w.Write(p.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		renderTemplate(w, view, p)
		return
	}
}

func cleaner(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Appengine-Cron") != "true" {
		http.Error(w, "Nope", http.StatusNotFound)
		return
	}

	c := appengine.NewContext(r)
	err := deleteOldPages(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("OK"))
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	body := r.FormValue("body")
	p := &Page{
		Time: time.Now(),
		Body: []byte(body),
	}
	id, err := p.save(c)
	if err != nil {
		c.Errorf("Error saving paste %d\n", id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.Debugf("Saving paste %v\n", id)
	http.Redirect(w, r, strconv.FormatInt(id, 10), http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates[tmpl].Execute(w, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
