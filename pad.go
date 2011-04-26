package main

import (
	"http"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
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
	http.ListenAndServe(":8080", nil)
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Id: "0", Body: make([]byte, 0)}
	renderTemplate(w, "paste", p)
}

func nextid() string {
	nxt := curid
	curid++
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
		templates[tmpl] = template.MustParseFile(tmpl+".html", nil)
	}
	os.Mkdir("pastes", 0755)

	curid = getlastid() + 1
	log.Println("Ready to serve")
}

func getlastid() int {
	p, err := os.Open("pastes")
	if err != nil {
		panic(err)
	}
	defer p.Close()

	names, err := p.Readdirnames(-1)
	if err != nil {
		panic(err)
	}

	nc := len(names)
	if nc == 0 {
		return 0
	}

	ids := make([]int, nc)
	for i := 0; i < nc; i++ {
		ids[i], err = strconv.Atoi(names[i])
		if err != nil {
			panic(err)
		}
	}
	sort.SortInts(ids)
	return ids[nc-1]
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
