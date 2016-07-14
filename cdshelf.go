package main

////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2015-2016, Gianluca Fiore
//
//    This program is free software: you can redistribute it and/or modify
//    it under the terms of the GNU General Public License as published by
//    the Free Software Foundation, either version 3 of the License, or
//    (at your option) any later version.
//
////////////////////////////////////////////////////////////////////////////////

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/shkh/lastfm-go/lastfm"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Configuration struct represents the data we need for authenticating on
// Last.fm
type Configuration struct {
	AuthorName string
	APIKey     string
	APISecret  string
	AppName    string
}

// AlbumPage struct represents the data that will appear on each album's page
type AlbumPage struct {
	Title	string
	Name	string
	Date	time.Time
	Cover	string
	Tags	[]string
	Content []byte
}

var artistArg string // artist name query
var albumArg string  // album name query

var usageMessage = `
cdshelf.go -a "<artist>" -l "<album>"

cdshelf.go look up for <artist> and <album> on Last.fm and return informations on the album

Arguments:

	-artist|-a <artist>
		Artist name. Enclose between "" if not a single word
	-album|-l <album>
		Album title. Enclose between "" if not a single word
`

// flagsInit initializes command line arguments
func flagsInit() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usageMessage)
	}

	const (
		defArtist = ""
		defAlbum  = ""
	)

	flag.StringVar(&artistArg, "artist", defArtist, "")
	flag.StringVar(&artistArg, "a", defArtist, "")
	flag.StringVar(&albumArg, "album", defAlbum, "")
	flag.StringVar(&albumArg, "l", defAlbum, "")

	flag.Parse()

	if artistArg == "" || albumArg == "" {
		fmt.Fprintf(os.Stderr, "You must give both artist and album names\n")
		os.Exit(2)
	}
}

// check is an helper function for handling errors
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// loadConfig loads the configuration file (in JSON)
func loadConfig() Configuration {
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("config.json doesn't exist")
		panic(err)
	}
	// make a new decoder and Configuration struct to host the config data
	decoder := json.NewDecoder(file)
	config := Configuration{}
	decodeErr := decoder.Decode(&config)
	if decodeErr != nil {
		fmt.Println("Couldn't parse config.json")
		panic(err)
	}
	return config
}

// getAuthorization authorizes the software on Last.fm
func getAuthorization(c Configuration) *lastfm.Api {
	if c.APIKey == "" || c.APISecret == "" {
		panic("You need an API Key and Secret in config.json. Please provide both, exiting...")
	}
	api := lastfm.New(c.APIKey, c.APISecret)
	token, err := api.GetToken()
	if err != nil {
		authURL := api.GetAuthTokenUrl(token)
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Authorize the cdshelf app at ", authURL)
		text, _ := reader.ReadString('\n')
		fmt.Println(text)
	}
	api.LoginWithToken(token)

	return api
}

// Print data received from Last.fm about an album. For debugging/helping
// printInfo is a debugging function that prints parts of the data received from 
// Last.fm about a specific album
func printInfo(info lastfm.AlbumGetInfo) {
	// Some calls:
	// Album name info.Name
	// Artist name info.Artist
	// Url info.Url
	// Release date info.ReleaseDate
	// Extralarge cover url info.Images[3].Url
	// Summary of wiki about the album info.Wiki.Summary
	// Wiki content about the album info.Wiki.Content
	fmt.Println(info.Name)
	fmt.Println(info.Artist)
	fmt.Println(info.Url)
	fmt.Println(info.Images[3].Url)
	fmt.Println(info.Wiki.Summary)
	fmt.Println(info.Wiki.Content)
}

// collectTags collects in a map the tags for a specific album as they are 
// received from Last.fm
func collectTags(tags lastfm.AlbumGetTopTags) map[string]string {
	m := make(map[string]string, 5)
	for i := 0; i <= 4; i++ {
		fmt.Println(tags.Tags[i].Name)
		fmt.Println(tags.Tags[i].Url)
		m[tags.Tags[i].Name] = tags.Tags[i].Url
	}

	return m
}

// downloadCover gets the album cover from Last.fm and saves it locally
func downloadCover(url, name string) {
	out, err := os.Create("static/images/" + name + ".png")
	defer out.Close()
	if err != nil {
		fmt.Println(err)
	}
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	_, errc := io.Copy(out, resp.Body)
	if errc != nil {
		fmt.Println(err)
	}
}

// save an AlbumPage content to a local file
func (a *AlbumPage) save() error {
	filename := a.Name + ".txt"
	return ioutil.WriteFile(filename, a.Content, 0600)
}

// load an album from local file to an AlbumPage struct
func (a *AlbumPage) load() (*AlbumPage, error) {
	filename := a.Name + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return &AlbumPage{Title: a.Title, Content: body}, nil
}

func main() {
	flagsInit()

	var tagMap map[string]string

	c := loadConfig()

	api := getAuthorization(c)

	info, err := api.Album.GetInfo(lastfm.P{
		"artist": artistArg,
		"album":  albumArg,
	})
	if err != nil {
		fmt.Println("Nothing found")
	} else {
		printInfo(info)
	}
	tags, err := api.Album.GetTopTags(lastfm.P{
		"artist": artistArg,
		"album":  albumArg,
	})
	if err != nil {
		fmt.Println(err)
		fmt.Println("No tags found")
	} else {
		tagMap = collectTags(tags)
		fmt.Println(len(tagMap))
	}

	downloadCover(info.Images[3].Url, info.Artist+"-"+info.Name)
}
