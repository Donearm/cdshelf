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

var artistArg string // artist name query
var albumArg string  // album name query

var usageMessage string = `
cdshelf.go -a "<artist>" -l "<album>"

cdshelf.go look up for <artist> and <album> on Last.fm and return informations on the album

Arguments:

	-artist|-a <artist>
		Artist name. Enclose between "" if not a single word
	-album|-l <album>
		Album title. Enclose between "" if not a single word
`

// Init command line arguments
func flagsInit() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usageMessage)
	}

	const (
		def_artist = ""
		def_album  = ""
	)

	flag.StringVar(&artistArg, "artist", def_artist, "")
	flag.StringVar(&artistArg, "a", def_artist, "")
	flag.StringVar(&albumArg, "album", def_album, "")
	flag.StringVar(&albumArg, "l", def_album, "")

	flag.Parse()

	if artistArg == "" || albumArg == "" {
		fmt.Fprintf(os.Stderr, "You must give both artist and album names\n")
		os.Exit(2)
	}
}

// Helper function for errors
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Load configuration file (in JSON)
func loadConfig() Configuration {
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("config.json doesn't exist")
		panic(err)
	}
	// make a new decoder and Configuration struct to host the config data
	decoder := json.NewDecoder(file)
	config := Configuration{}
	decode_err := decoder.Decode(&config)
	if decode_err != nil {
		fmt.Println("Couldn't parse config.json")
		panic(err)
	}
	return config
}

// Authorize on Last.fm
func getAuthorization(c Configuration) *lastfm.Api {
	if c.APIKey == "" || c.APISecret == "" {
		panic("You need an API Key and Secret in config.json. Please provide both, exiting...")
	}
	api := lastfm.New(c.APIKey, c.APISecret)
	token, err := api.GetToken()
	if err != nil {
		authUrl := api.GetAuthTokenUrl(token)
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Authorize the cdshelf app at ", authUrl)
		text, _ := reader.ReadString('\n')
		fmt.Println(text)
	}
	api.LoginWithToken(token)

	return api
}

// Print data received from Last.fm about an album. For debugging/helping
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

// Collect in a map tags for a specific album as they are on Last.fm
func collectTags(tags lastfm.AlbumGetTopTags) map[string]string {
	m := make(map[string]string, 5)
	for i := 0; i <= 4; i++ {
		fmt.Println(tags.Tags[i].Name)
		fmt.Println(tags.Tags[i].Url)
		m[tags.Tags[i].Name] = tags.Tags[i].Url
	}

	return m
}

// Get album cover and save it locally
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

// Write info from Last.fm to a markdown file
func writeMarkdown(name, title, summary, content string, tags map[string]string) {
	// create file for the album. Replace all whitespaces with _
	entry, err := os.Create("content/album/" + strings.Replace(name, " ", "_", -1) + "-" + strings.Replace(title, " ", "_", -1) + ".md")
	defer entry.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Generate time string
	t := time.Now()

	// Start progressively writing the toml header
	_, e1 := entry.WriteString("+++\ndate = \"" + t.Format("2006-01-02T15:04:05Z") + "\"\n")
	check(e1)
	_, e2 := entry.WriteString("name = \"" + name + " - " + title + "\"\n")
	check(e2)
	_, e3 := entry.WriteString("tags = " + "[\n")
	check(e3)
	for k, _ := range tags {
		_, e := entry.WriteString("\t\"" + k + "\",\n")
		check(e)
	}
	_, e4 := entry.WriteString("]\n")
	check(e4)
	_, e5 := entry.WriteString("title = \"" + title + "\"\n")
	check(e5)
	_, e6 := entry.WriteString("image = \"" + name + "-" + title + ".png\"\n")
	check(e6)
	_, e7 := entry.WriteString("\n+++\n")
	check(e7)
	if content != "" {
		_, e8 := entry.WriteString(content)
		check(e8)
	} else {
		_, e8 := entry.WriteString(summary)
		check(e8)
	}
	entry.Sync()
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

	writeMarkdown(info.Artist, info.Name, info.Wiki.Summary, info.Wiki.Content, tagMap)
}
