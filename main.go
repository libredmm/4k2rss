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

func scrape_thread(href string) feeds.Item {
	thread_url, err := url.JoinPath(BASE_URL, href)
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.Get(thread_url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	enclosure_url, err := url.JoinPath(
		BASE_URL,
		doc.Find("ul.attachlist a[href^='attach-download']").AttrOr("href", ""))
	if err != nil {
		log.Fatal(err)
	}
	return feeds.Item{
		Title:       doc.Find("title").Text(),
		Link:        &feeds.Link{Href: thread_url},
		Description: doc.Find("div.message").Text(),
		Enclosure:   &feeds.Enclosure{Url: enclosure_url, Length: "0", Type: "application/x-bittorrent"},
	}
}

func scrape_forum_page(items chan<- feeds.Item, category int, page int) {
	log.Printf("Forum: category=%d, page=%d", category, page)
	forum_url := fmt.Sprintf("https://4k2.com/forum-%d-%d.htm?orderby=tid", category, page)
	res, err := http.Get(forum_url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	doc.Find("ul.threadlist li.thread div.media-body div.style3_subject a[href^='thread-']").Each(
		func(i int, s *goquery.Selection) {
			thread_url, exists := s.Attr("href")
			if exists {
				wg.Add(1)
				go func() {
					defer wg.Done()
					items <- scrape_thread(thread_url)
				}()
			}
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
		log.Printf("Thread: %s", item.Title)
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
