package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
)

const BASE_URL = "https://4k2.com/"

func get_and_parse(u string) *goquery.Document {
	res, err := http.Get(u)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func scrape_thread(href string) feeds.Item {
	thread_url, _ := url.JoinPath(BASE_URL, href)
	log.Println(thread_url)
	doc := get_and_parse(thread_url)
	enclosure_url, _ := url.JoinPath(
		BASE_URL,
		doc.Find("ul.attachlist a[href^='attach-download']").AttrOr("href", ""))
	return feeds.Item{
		Title:       doc.Find("title").Text(),
		Link:        &feeds.Link{Href: thread_url},
		Description: doc.Find("div.message").Text(),
		Enclosure:   &feeds.Enclosure{Url: enclosure_url, Length: "0", Type: "application/x-bittorrent"},
	}
}

func scrape_forum_page(items chan<- feeds.Item, category int, page int) {
	forum_url := fmt.Sprintf("https://4k2.com/forum-%d-%d.htm?orderby=tid", category, page)
	log.Println(forum_url)
	doc := get_and_parse(forum_url)
	var wg sync.WaitGroup
	doc.Find("ul.threadlist li.thread div.media-body div.style3_subject a[href^='thread-']").Each(
		func(i int, s *goquery.Selection) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				items <- scrape_thread(s.AttrOr("href", ""))
			}()
		})
	wg.Wait()
}

func scrape_forum(id int, max_page int, title string) {
	items := make(chan feeds.Item)
	var wg sync.WaitGroup
	for page := 1; page <= max_page; page++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			scrape_forum_page(items, id, p)
		}(page)
	}
	go func() {
		wg.Wait()
		close(items)
	}()
	feed := &feeds.Feed{
		Title:       title,
		Link:        &feeds.Link{Href: fmt.Sprintf("https://4k2.com/forum-%d-1.htm?orderby=tid", id)},
		Description: title,
		Created:     time.Now(),
	}
	for item := range items {
		feed.Add(&item)
	}
	feed.Sort(func(a, b *feeds.Item) bool {
		return a.Link.Href > b.Link.Href
	})
	rss, err := feed.ToRss()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rss)
}

func main() {
	scrape_forum(1, 1, "亚洲有码-4K2社区")
}
