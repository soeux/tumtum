package downloader

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soeux/tumtum/config"
	"github.com/soeux/tumtum/database"
	"github.com/soeux/tumtum/scraper"
	"github.com/urfave/cli/v2"
)

// gets everything started
func HandleLink(c *cli.Context, url string) error {
	// creating parentContext
	ctx := parentContext()

	cfg, err := config.LoadConfigOrDefault("tumtum.toml")
	if err != nil {
		return err
	}

	db, err := database.NewDB()
	if err != nil {
		return err
	}

	httpClient := newHTTPClient() // newHTTPClient(jar)

	s := scraper.NewScraper(httpClient, db)

	times, err := s.Scrape(ctx, url, cfg)
	if err != nil {
		if !isContextCanceledError(err) {
			log.Println(err)
		}
		return err
	}

	err = db.SetTime(url, times)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// im still 100% sure what this does, but it might be part of the problem in the source.
func parentContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		defer signal.Stop(ch)

		<-ch
	}()

	return ctx
}

// func newHTTPClient(jar *cookiejar.Jar) *http.Client {
func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 60 * time.Second,
		Jar:     nil, // hopefully this works bc i don't want to implement cookies
	}
}

func isContextCanceledError(err error) bool {
	if e, ok := err.(*url.Error); ok {
		err = e.Err
	}

	return err == context.Canceled
}
