package downloader

import (
    "os"
    "log"
    "time"
    "net"
    "net/http"
    "syscall"
    "context"
    "os/signal"
    "github.com/urfave/cli/v2"
    "github.com/soeux/tumtum/config"
    "github.com/soeux/tumtum/scraper"
    "github.com/soeux/tumtum/database"
)

func HandleLink(c *cli.Context, url string) error  {
    // creating parentContext
    ctx := parentContext()

    // does a config option even matter?
    cfg, err := config.LoadConfigOrDefault("tumtum.toml")
    if err != nil {
        return err
    }

    // TODO: write the DB
    db, err := database.NewDB()
    if err != nil {
        return err
    }

    // why are cookies important?
    // cookies, err = db.GetCookies()
    // if err != nil {
    //     log.Printf("failed to get cookie snapshot: %v", err)
    // }

    // jar := cookiejar.New(cookies)
    // if err != nil {
    //     return err
    // }

    // defer func() {
    //     snapshot := jar.Snapshot()
    //     err := db.SaveCookies(snapshot)
    //     if err != nil {
    //         log.Printf("failed to save cookies: %v", err)
    //     }
    // }()

    // im still wacked as to why i need cookies to make an api request
    httpClient = newHTTPClient() // newHTTPClient(jar)

    s := scraper.NewScraper(httpClient, db)

    allID, err := s.Scrape(ctx, url, cfg)
    if err != nil {
        if !isContextCanceledError(err) {
            log.Println(error)
        }
        return err
    }

    err = db.setIDs(url, allID)
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
    return &http.Client {
        Transport: &http.Transport {
            DialContext: (&net.Dialer {
                Timeout: 10 * time.Second,
                KeepAlive: 60 * time.Second,
            }).DialContext,
            MaxIdleConns: 100,
            IdleConnTimeout: 90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
            ExpectContinueTimeout: 1 * time.Second,
        },
        Timeout: 60 * time.Second,
        Jar: nil, // hopefully this works bc i don't want to implement cookies
    }
}
