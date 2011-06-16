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

var maxPasteLen = 4096
var templates = make(map[string]*template.Template)
var templmtimes = make(map[string]int64)
var idValidator = regexp.MustCompile("^[0-9]+$")

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
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	p := new(Page)
	if len(id) > 0 {
		ids, err := strconv.Atoi64(id)
		if err == nil {
			p, err = loadPage(r, ids)
			if err != nil {
				// Oh well, give them a blank paste
			}
		}
	}
	renderTemplate(w, "paste", p)
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

func makeHandler(fn func(http.ResponseWriter, *http.Request, int64, string), tmplt string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len(tmplt)+2:]
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

func mtime(f string) int64 {
	fi, err := os.Stat(f)
	if err != nil {
		return 0
	}
	return fi.Mtime_ns
}
