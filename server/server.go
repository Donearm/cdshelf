package server

////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2016, Gianluca Fiore
//
//    This program is free software: you can redistribute it and/or modify
//    it under the terms of the GNU General Public License as published by
//    the Free Software Foundation, either version 3 of the License, or
//    (at your option) any later version.
//
////////////////////////////////////////////////////////////////////////////////


import (
	"fmt"
	"net/http"
	"io/ioutil"
)

type AlbumFile struct {
	Title	string
	Body	[]byte
}

func loadAlbum(title string) (*AlbumFile, error) {
	body, err := ioutil.ReadFile(title)
	if err != nil {
		return nil, err
	}

	return &AlbumFile{Title: title, Body: body}, nil
}

func albumHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/album/"):]
	p, _ := loadAlbum(title)
	fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Test %s", r.URL.Path[1:])
}

