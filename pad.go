// © 2018 Steve McCoy. Available under the MIT License.

package main

import (
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"text/template"
	"time"
)

var maxPasteLen = 1048576
var templates = make(map[string]*template.Template)
var viewValidator = regexp.MustCompile("^/([0-9a-z]+)(/([a-z]+)?)?$")

var icons = []string{
	"📋",
	"🗒",
}

func main() {
	rand.Seed(3)

	for _, tmpl := range []string{"paste", "plain", "wrapped"} {
		t := "tmplt/" + tmpl + ".html"
		templates[tmpl] = template.Must(template.ParseFiles(t))
	}

	http.Handle("/css/", 
		http.StripPrefix("/css/", http.FileServer(http.Dir("./css"))))
	http.Handle("/js/",
		http.StripPrefix("/js/", http.FileServer(http.Dir("./js"))))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request){
		http.ServeFile(w, r, "./favicon.ico")
	})
	http.HandleFunc("/", pasteHandler)

	http.ListenAndServe(":9457", nil)
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("path = %s", r.URL.Path)

	if r.Method == "POST" && r.URL.Path == "/" {
		log.Println("posting")
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

	id, err := strconv.ParseInt(parts[1], 36, 64)
	if err != nil {
		log.Printf("id = %s, err = %v", parts[1], err)
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
		if err != nil || view != "plain" && view != "raw" && view != "wrapped" {
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

func saveHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	body := r.FormValue("body")
	p := &Page{
		Time: time.Now(),
		Body: []byte(body),
	}
	id, err := p.save(db)
	if err != nil {
		log.Printf("Error saving paste %d", id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Saving paste %v", id)
	http.Redirect(w, r, strconv.FormatInt(id, 36), http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	encPage := struct {
		Icon string
		Id   string
		Body []byte
	}{
		Icon: icons[rand.Intn(len(icons))],
		Id:   strconv.FormatInt(p.Id, 36),
		Body: p.Body,
	}
	err := templates[tmpl].Execute(w, encPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
