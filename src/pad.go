package main

import (
	"appengine"
	"appengine/datastore"
	"http"
	"os"
	"regexp"
	"strconv"
	"template"
	"time"
)

type Page struct {
	Id int64
	Time datastore.Time
	Body []byte
}

func (p *Page) save(c appengine.Context) (id int64, err os.Error) {
	if len(p.Body) > 4096 {
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

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Time: 0, Body: make([]byte, 0)}
	renderTemplate(w, "paste", p)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, int64, string), tmplt string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/"+tmplt+"/"):]
		if !idValidator.MatchString(id) {
			http.NotFound(w, r)
			return
		}
		i, err := strconv.Atoi64(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, i, tmplt)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, id int64, tmplt string) {
	p, err := loadPage(r, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, tmplt, p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	body := r.FormValue("body")
	p := &Page{
		Time: datastore.SecondsToTime(time.Seconds()),
		Body: []byte(body),
	}
	id, err := p.save(c)
	if err != nil {
		c.Logf("Error saving paste %s\n", id)
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	c.Logf("Saving paste %v\n", id)
	http.Redirect(w, r, "/plain/"+strconv.Itoa64(id), http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	t := "tmplt/" + tmpl + ".html"
	mt := mtime(t)
	if mt > templmtimes[tmpl] {
		templmtimes[tmpl] = mt
		templates[tmpl] = template.MustParseFile(t, nil)
	}
	err := templates[tmpl].Execute(w, p)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
}

var templates = make(map[string]*template.Template)
var templmtimes = make(map[string]int64)

func init() {
	for _, tmpl := range []string{"paste", "plain", "fancy"} {
		t := "tmplt/" + tmpl + ".html"
		templmtimes[tmpl] = mtime(t)
		templates[tmpl] = template.MustParseFile(t, nil)
	}

	http.HandleFunc("/", pasteHandler)
	http.HandleFunc("/plain/", makeHandler(viewHandler, "plain"))
        http.HandleFunc("/fancy/", makeHandler(viewHandler, "fancy"))
        http.HandleFunc("/save/", saveHandler)
        http.Handle("/js/", http.FileServer("js/", "/js/"))
        http.Handle("/css/", http.FileServer("css/", "/css/"))
}

func mtime(f string) int64 {
	fi, err := os.Stat(f)
	if err != nil {
		return 0
	}
	return fi.Mtime_ns
}

var idValidator = regexp.MustCompile("^[0-9]+$")
