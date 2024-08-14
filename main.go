package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3_types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gorilla/feeds"
)

const BASE_URL = "https://4k2.com/"

func get_full_url(path string) string {
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = BASE_URL
	}
	full_url, _ := url.JoinPath(base, path)
	return full_url
}

var proxyTransport = &http.Transport{
	Proxy:           http.ProxyFromEnvironment,
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
var httpClient = http.Client{
	Transport: proxyTransport,
}

func get_and_parse(u string) *goquery.Document {
	log.Printf("GET %s", u)
	res, err := httpClient.Get(u)
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
	thread_url := get_full_url(href)
	doc := get_and_parse(thread_url)
	enclosure_url := get_full_url(
		doc.Find("ul.attachlist a[href^='attach-download']").AttrOr("href", ""),
	)
	item := feeds.Item{
		Title:       doc.Find("title").Text(),
		Link:        &feeds.Link{Href: thread_url},
		Description: doc.Find("div.message").Text(),
		Enclosure:   &feeds.Enclosure{Url: enclosure_url, Length: "0", Type: "application/x-bittorrent"},
	}
	log.Printf("Item: %s", item.Title)
	return item
}

func scrape_forum_page(items chan<- feeds.Item, category int, page int) {
	forum_url := get_full_url(
		fmt.Sprintf("forum-%d-%d.htm?orderby=tid", category, page),
	)
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

func scrape_forum(id int, max_page int, title string, s3_path string) {
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
		Link:        &feeds.Link{Href: get_full_url(fmt.Sprintf("forum-%d-1.htm?orderby=tid", id))},
		Description: title,
		Created:     time.Now(),
	}
	for item := range items {
		if item.Title != "" {
			feed.Add(&item)
		}
	}
	feed.Sort(func(a, b *feeds.Item) bool {
		return a.Link.Href > b.Link.Href
	})
	rss, err := feed.ToRss()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("RSS generated, category: %d, items#: %d", id, len(feed.Items))
	aws_cfg, err := aws_config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	s3_client := s3.NewFromConfig(aws_cfg)
	_, err = s3_client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String("libredmm"),
		Key:         &s3_path,
		ACL:         s3_types.ObjectCannedACLPublicRead,
		Body:        strings.NewReader(rss),
		ContentType: aws.String("application/rss+xml"),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Uploaded to s3: %s", s3_path)
}

func scrape(pages int, interval int, dryrun bool) {
	if dryrun {
		log.Println("Dry run")
	} else {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			scrape_forum(1, pages, "4K2社区-亚洲有码", "feeds/4k2/hd.xml")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			scrape_forum(3, pages, "4K2社区-4K专区", "feeds/4k2/4k.xml")
		}()
		wg.Wait()
	}
	if interval > 0 {
		log.Printf("Sleeping for %d seconds before next run", interval)
		time.Sleep(time.Duration(interval) * time.Second)
		scrape(pages, interval, dryrun)
	}
}

func main() {
	pages := flag.Int("pages", 3, "Scrape first N pages")
	interval := flag.Int("interval", 0, "Interval in seconds for repeated scraping, 0 for no repeat")
	dryrun := flag.Bool("dryrun", false, "Dry run")
	flag.Parse()
	scrape(*pages, *interval, *dryrun)
}
