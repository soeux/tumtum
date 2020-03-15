package main

import (
    "os"
    "fmt"
    "time"
    "strings"
    "github.com/urfave/cli/v2"
    "github.com/soeux/tumtum/downloader"
)

func main() {
    var blogURL string

    app := &cli.App {
        Name: "tumtum",
        Version: "v0.1",
        Compiled: time.Now(),
        Usage: "downloads all media from tumblr blog",
        Flags: []cli.Flag {
            &cli.StringFlag {
                Name: "download",
                Aliases: []string{"d"},
                Usage: "downloads from blog `URL`",
                Required: true,
                Destination: &blogURL,
            },
        },
        Action: func(c *cli.Context) error {
            if blogURL != "" {
                // check if it's a tumblr link
                if strings.Contains(blogURL, ".tumblr.com") {
                    downloader.HandleLink(c, blogURL)
                } else if strings.ContainsRune(blogURL, '.') && !strings.Contains(blogURL, ".tumblr.com") {
                    // probably a custom domain
                    downloader.HandleLink(c, blogURL)
                } else {
                    downloader.HandleLink(c, blogURL + ".tumblr.com")
                }
            }

            return nil
        },
    }

    err := app.Run(os.Args)
    if err != nil {
        fmt.Fprintln(cli.ErrWriter, err)
        cli.OsExiter(1)
    }
}
