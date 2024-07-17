package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	tfm := template.FuncMap{
		"ts": func(u int64) string {
			return time.Unix(u, 0).Format(time.RFC822)
		},
		"spacer": func(l uint) uint {
			return 20 * l
		},
	}

	tmpl, err := template.New("who-is-hiring.tmpl.html").Funcs(tfm).ParseFiles("who-is-hiring.tmpl.html")
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	fmt.Println("Getting story ID...")
	storyID, err := getStoryID()
	if err != nil {
		log.Fatalf("Error getting story ID: %v", err)
	}
	fmt.Printf("Got story ID (%d).\n", storyID)

	fmt.Println("Getting story...")
	story, err := getStory(storyID)
	if err != nil {
		log.Fatalf("Error getting story %d: %v", storyID, err)
	}
	fmt.Println("Got story.")

	const filename string = "index.html"
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Error creating %s: %v", filename, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, story); err != nil {
		log.Fatalf("Error executing template: %v", err)
	}
}

const searchURL string = "https://hn.algolia.com/api/v1/search?query=%%22ask%%20hn:%%20who%%20is%%20hiring%%3F%%20(%s)%%22"

type algoliaRes struct {
	NbHits int `json:"nbHits"`
	Hits   []struct {
		StoryID uint `json:"story_id"`
	}
}

func getStoryID() (uint, error) {
	now := time.Now()
	curMon := now.Format("January 2006")
	lastMon := now.AddDate(0, -1, 0).Format("January 2006")

	for _, mon := range []string{curMon, lastMon} {
		res, err := http.Get(fmt.Sprintf(searchURL, url.QueryEscape(mon)))
		if err != nil {
			return 0, fmt.Errorf("error searching Algolia for article in %s: %w", mon, err)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return 0, fmt.Errorf("error reading Algolia res body for article in %s: %w", mon, err)
		}

		var data algoliaRes
		if err := json.Unmarshal(body, &data); err != nil {
			return 0, fmt.Errorf("error unmarshaling Algolia res data for article in %s: %w", mon, err)
		}

		if data.NbHits > 0 {
			return data.Hits[0].StoryID, nil
		}
	}

	return 0, fmt.Errorf("could not find story for %s or %s", curMon, lastMon)
}

const itemURL string = "https://hacker-news.firebaseio.com/v0/item/%d.json?print=pretty"

func getItem(id uint) ([]byte, error) {
	const maxTries int = 3 // Because the API can be a bit flaky, we will try a few times.

	for i := 0; i < maxTries; i++ {
		res, err := http.Get(fmt.Sprintf(itemURL, id))
		if err != nil {
			log.Printf("error getting item %d from Firebase API: %v", id, err)
			continue
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Printf("error reading item %d Firebase API res body: %v", id, err)
			continue
		}

		return body, nil
	}

	return nil, fmt.Errorf("could not get item %d after %d tries", id, maxTries)
}

type comment struct {
	ID      uint `json:"id"`
	Time    int64
	Text    template.HTML
	By      string
	Dead    bool
	Deleted bool
	Kids    []uint
	Level   uint
	Remote  bool
	Interns bool
	Visa    bool
}

type story struct {
	ID        uint `json:"id"`
	Title     string
	Text      template.HTML
	By        string
	Kids      []uint
	Comments  []comment
	FetchedAt int64
}

func getComments(id, level uint) ([]comment, error) {
	commentJSON, err := getItem(id)
	if err != nil {
		return nil, fmt.Errorf("error getting comment %d JSON: %w", id, err)
	}

	var c comment
	if err := json.Unmarshal(commentJSON, &c); err != nil {
		return nil, fmt.Errorf("error unmarshaling comment %d JSON: %w", id, err)
	}

	if c.Dead || c.Deleted {
		return nil, nil
	}

	lowText := strings.ToLower(string(c.Text))
	if strings.Contains(lowText, "remote") {
		c.Remote = true
	}
	if strings.Contains(lowText, "interns") {
		c.Interns = true
	}
	if strings.Contains(lowText, "visa") {
		c.Visa = true
	}

	c.Level = level

	var comments = make([]comment, 0, len(c.Kids)+1)
	comments = append(comments, c)

	for _, kid := range c.Kids {
		sc, err := getComments(kid, level+1)
		if err != nil {
			return nil, fmt.Errorf("error getting subcomments rooted at %d: %w", kid, err)
		}

		comments = append(comments, sc...)
	}

	return comments, nil
}

func getStory(id uint) (story, error) {
	now := time.Now()

	storyJSON, err := getItem(id)
	if err != nil {
		return story{}, fmt.Errorf("error getting story %d JSON: %w", id, err)
	}

	var s story
	if err := json.Unmarshal(storyJSON, &s); err != nil {
		return story{}, fmt.Errorf("error unmarshaling story %d JSON: %w", id, err)
	}

	for i, kid := range s.Kids {
		if i%25 == 0 {
			fmt.Printf("Getting top-level comment %d...\n", i+1)
		}
		cs, err := getComments(kid, 0)
		if err != nil {
			return story{}, fmt.Errorf("error getting story comments rooted at %d: %w", kid, err)
		}

		s.Comments = append(s.Comments, cs...)
	}

	s.FetchedAt = now.Unix()

	return s, nil
}
