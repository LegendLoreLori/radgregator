package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/LegendLoreLori/radgregator/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

var commonDateLayouts = []string{time.RFC1123, time.RFC1123Z, time.RFC3339, time.RFC3339Nano, time.RFC822, time.RFC822Z, time.RFC850}

func fetchFeed(ctx context.Context, feedUrl string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "radgregator")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	feedData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var feed RSSFeed
	err = xml.Unmarshal(feedData, &feed)
	if err != nil {
		return nil, err
	}
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i := 0; i < len(feed.Channel.Item); i++ {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil
}

func scrapeFeeds(ctx context.Context, s *state) error {
	feedDetails, err := s.db.GetNextFeedToFetch(ctx)
	if err != nil {
		return err
	}
	err = s.db.MarkFeedFetched(ctx, database.MarkFeedFetchedParams{
		UpdatedAt: time.Now(),
		ID:        feedDetails.ID,
	})
	if err != nil {
		return err
	}

	feed, err := fetchFeed(ctx, feedDetails.Url)
	if err != nil {
		return err
	}

	fmt.Printf("saving posts for %s...", feed.Channel.Title)
	for _, post := range feed.Channel.Item {
		pubDate := sql.NullTime{}
		title := sql.NullString{}
		description := sql.NullString{}

		for _, format := range commonDateLayouts {
			t, err := time.Parse(format, post.PubDate)
			if err != nil {
				continue
			}
			pubDate = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		}
		if len(post.Title) != 0 {
			title = sql.NullString{
				String: post.Title,
				Valid:  true,
			}
		}
		if len(post.Description) != 0 {
			title = sql.NullString{
				String: post.Description,
				Valid:  true,
			}
		}

		_, err = s.db.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       title,
			Url:         post.Link,
			Description: description,
			PublishedAt: pubDate,
			FeedID:      feedDetails.ID,
		})
		var sqlErr *pq.Error
		if errors.As(err, &sqlErr) {
			if sqlErr.Code == "23505" { // duplicate key entry
				continue
			}
			log.Print(err)
		}
	}
	println("done")
	return nil
}
