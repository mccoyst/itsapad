package main

import (
	"fmt"
	"http"
)

func main() {
	http.HandleFunc("/", index)
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>Paste to the pad.</h1><p>%s</p>", "please")
}
