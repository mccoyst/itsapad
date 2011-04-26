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

const lenPath = len("/view/")

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
	http.HandleFunc("/view/", makeHandler(viewHandler))
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
		writeNextId(id+1)
		idchan <- id
	}
}

func readNextId() int {
	ids, err := ioutil.ReadFile("pastes/next")
	if err != nil {
		panic(err)
	}
	id, err := strconv.Atoi(strings.TrimSpace(string(ids)))
	if err != nil {
		panic(err)
	}
	return id
}

func writeNextId(id int) {
	ids := strconv.Itoa(id)
	bytes := make([]byte, len(ids))
	copy(bytes, ids)
	err := ioutil.WriteFile("pastes/next", bytes, 0600)
	if err != nil {
		panic(err)
	}
}

func nextid() string {
	servchan <- 1
	nxt := <-idchan
	return strconv.Itoa(nxt)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Path[lenPath:]
		if !titleValidator.MatchString(title) {
			http.NotFound(w, r)
			return
		}
		fn(w, r, title)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "view", p)
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
	http.Redirect(w, r, "/view/"+id, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates[tmpl].Execute(w, p)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
}

var templates = make(map[string]*template.Template)
var curid int

func init() {
	log.Println("Starting up")
	for _, tmpl := range []string{"paste", "view"} {
		templates[tmpl] = template.MustParseFile("tmplt/"+tmpl+".html", nil)
	}
	os.Mkdir("pastes", 0755)

	go idsrv()
	log.Println("Ready to serve")
}

var titleValidator = regexp.MustCompile("^[0-9]+$")

func getId(w http.ResponseWriter, r *http.Request) (title string, err os.Error) {
	title = r.URL.Path[lenPath:]
	if !titleValidator.MatchString(title) {
		http.NotFound(w, r)
		err = os.NewError("Invalid Page Id")
	}
	return
}
