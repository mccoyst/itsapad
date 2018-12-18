// © 2018 Steve McCoy. Available under the MIT License.

package main

import (
	"database/sql"
	"bytes"
	"errors"
	"net/http"
	"time"

	_ "rsc.io/sqlite"
)

type Page struct {
	Id   int64
	Time time.Time
	Body []byte
}

func (p *Page) save(db *sql.DB) (id int64, err error) {
	if len(p.Body) > maxPasteLen {
		err = errors.New("Paste is too large to store")
		return
	}
	p.Body = bytes.Replace(p.Body, []byte{'\r'}, []byte{}, -1)
	r, err := db.Exec("insert into pastes (time, body) values (?, ?)", time.Now().Unix(), p.Body)
	if err != nil {
		return
	}
	return r.LastInsertId()
}

func loadPage(r *http.Request, id int64) (*Page, error) {
	db, err := connectDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var t int64
	var body []byte
	row := db.QueryRow("select time, body from pastes where rowid = ?", id)
	err = row.Scan(&t, &body)
	if err != nil {
		return nil, err
	}
	return &Page{Id: id, Time: time.Unix(t, 0), Body: body}, nil
}

func connectDB() (*sql.DB, error) {
	return sql.Open("sqlite3", "./pastes.db")
}
