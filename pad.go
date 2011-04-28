package main

import (
	"http"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"template"
)

type Page struct {
	Id   string
	Body []byte
}

func (p *Page) save() os.Error {
	filename := "pastes/" + p.Id
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, os.Error) {
	filename := "pastes/" + title
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Id: title, Body: body}, nil
}

func main() {
	http.HandleFunc("/", pasteHandler)
	http.HandleFunc("/plain/", makeHandler(viewHandler, "plain"))
	http.HandleFunc("/fancy/", makeHandler(viewHandler, "fancy"))
	http.HandleFunc("/save/", saveHandler)
	http.Handle("/js/", http.FileServer("js/", "/js/"))
	http.Handle("/css/", http.FileServer("css/", "/css/"))
	http.ListenAndServe(":8080", nil)
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Id: "0", Body: make([]byte, 0)}
	renderTemplate(w, "paste", p)
}

var idchan = make(chan int)
var servchan = make(chan int)

func idsrv() {
	for {
		<-servchan
		id := readNextId()
		writeNextId(id + 1)
		idchan <- id
	}
}

func readNextId() int {
	ids, err := ioutil.ReadFile("pastes/next")
	if err != nil {
		return 0
	}
	id, err := strconv.Atoi(strings.TrimSpace(string(ids)))
	giveUpOn(err)
	return id
}

func writeNextId(id int) {
	bytes := []byte(strconv.Itoa(id))
	err := ioutil.WriteFile("pastes/next", bytes, 0600)
	giveUpOn(err)
}

func nextid() string {
	servchan <- 1
	nxt := <-idchan
	return strconv.Itoa(nxt)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string, string), tmplt string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Path[len("/"+tmplt+"/"):]
		if !titleValidator.MatchString(title) {
			http.NotFound(w, r)
			return
		}
		fn(w, r, title, tmplt)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string, tmplt string) {
	p, err := loadPage(title)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, tmplt, p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	body := r.FormValue("body")
	id := nextid()
	p := &Page{Id: id, Body: []byte(body)}
	err := p.save()
	if err != nil {
		log.Printf("Error saving paste %s\n", id)
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	log.Printf("Saving paste %s\n", id)
	http.Redirect(w, r, "/plain/"+id, http.StatusFound)
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
var curid int

func init() {
	log.Println("Starting up")
	for _, tmpl := range []string{"paste", "plain", "fancy"} {
		t := "tmplt/" + tmpl + ".html"
		templmtimes[tmpl] = mtime(t)
		templates[tmpl] = template.MustParseFile(t, nil)
	}
	os.Mkdir("pastes", 0755)

	go idsrv()
	log.Println("Ready to serve")
}

func mtime(f string) int64 {
	fi, err := os.Stat(f)
	giveUpOn(err)
	return fi.Mtime_ns
}

var titleValidator = regexp.MustCompile("^[0-9]+$")

func giveUpOn(err os.Error) {
	if err != nil {
		panic(err)
	}
}
