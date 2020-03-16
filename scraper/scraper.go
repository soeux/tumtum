package scraper

import (
    "os"
    "io"
    "io/ioutil"
    "fmt"
    "log"
    // "net"
    "net/url"
    "net/http"
    "math"
    "mime"
    "time"
    "regexp"
    // "strconv"
    "strings"
    "errors"
    "context"
    "encoding/json"
    "path/filepath"
    // "golang.org/x/net/html"
    // "golang.org/x/net/html/atom"
    "golang.org/x/sync/errgroup"
    "github.com/soeux/tumtum/config"
    "github.com/soeux/tumtum/database"
    "github.com/soeux/tumtum/semaphore"
)

var (
    errFileNotFound = errors.New("file not found")

    deactivatedNameSuffixLength = 20
    deactivatedNameRegexp = regexp.MustCompile(`.-deactivated\d{8}$`)

    videoURLFixupRegexp = regexp.MustCompile(`_(?:480|720)\.mp4$`)
    imageSizeFixupRegexp = regexp.MustCompile(`_(?:\d+)\.([a-z]+)$`)

    mediaURLRegexp = regexp.MustCompile(`^http.+(?:media|vtt)\.tumblr\.com/.+$`)
    htmlMediaURLRegexp = regexp.MustCompile(`http[^"]+(?:media|vtt)\.tumblr\.com/[^"]+`)
)

func init() {
    for _, e := range []struct{typ, ext string} {
        {"image/bmp", ".bmp"},
        {"image/gif", ".gif"},
        {"image/jpeg", ".jpg"},
        {"image/png", ".png"},
        {"image/tiff", ".tiff"},
        {"image/webp", ".webp"},
        {"video/webm", ".webm"},
    } {
        err := mime.AddExtensionType(e.ext, e.typ)
        if err != nil {
            panic(err)
        }
    }
}

// scraper obkect
type Scraper struct {
    client *http.Client
    config *config.Config
    databse *database.Database
}

// initalising a scraper obj
func NewScraper(client *http.Client, database *database.Database) *Scraper {
    return &Scraper {
        client: client,
        databse: database,
    }
}

// creating the save location + starting a child process for scraper
func (s *Scraper) Scrape(ctx context.Context, link string, cfg *config.Config) (map[string]int64, error) {
    err := os.MkdirAll(cfg.Save, 0755)
    if err != nil {
        return map[string]int64{"0":0}, err
    }

    // are we creating a child process here?
    eg, ctx := errgroup.WithContext(ctx)

    sc := newScrapeContext(s, cfg, link, eg, ctx)
    if err != nil {
        return map[string]int64{"0":0}, err
    }

    err = sc.Scrape()
    if err != nil {
        return map[string]int64{"0":0}, err
    }

    return sc.ids, nil
}

type scrapeContextState int

// ????
const (
    scrapeContextStateTryUseAPI scrapeContextState = iota
    scrapeContextStateUseAPI
    scrapeContextStateTryUseIndashAPI
    scrapeContextStateUseIndashAPI
)

type scrapeContext struct {
    // structuralised arguments
    scraper *Scraper
    config *config.Config
    link string
    errgroup * errgroup.Group
    ctx context.Context

    // general state of this scrapeContext
    state scrapeContextState

    // current pagination state
    offset int
    before time.Time

    // informational values
    // highest_id, lowest_id, current_id
    ids map[string]int64

    // other private members
    sema *semaphore.PrioritySemaphore
}

func newScrapeContext(s *Scraper, cfg *config.Config, link string, eg *errgroup.Group, ctx context.Context) *scrapeContext {
    // initalising a scrapeContext
    sc := &scrapeContext {
        scraper: s,
        config: cfg,
        link: link,
        errgroup: eg,
        ctx: ctx,
        state: scrapeContextStateTryUseAPI,
        ids: map[string]int64{
            "highest_id": math.MinInt64,
            "lowest_id": math.MaxInt64,
            "current_id": 0,
        },
        sema: semaphore.NewPrioritySemaphore(s.config.Concurrency),
    }

    return sc
}

func (sc *scrapeContext) Scrape() (err error) {
    log.Printf("%s: scraping starting at %d", sc.link, sc.ids["highest_id"])
    defer func() {
        log.Printf("%s: scraping finished at %d", sc.link, sc.ids["highest_id"])
    }()

    defer func() {
        e := sc.errgroup.Wait()
        if err != nil {
            err = e
        }
    }()

    initHighestID := sc.ids["highest_id"]

    for {
        // figure out how to implement the ID thing

        var res *postsResponse
        res, err = sc.scrapeBlog()
        if err != nil {
            return
        }

        // no posts
        if len(res.Response.Posts) == 0 {
            return
        }

        // convert postID to int64
        for _, post := range res.Response.Posts {
            post.id, err = post.ID.Int64()
            if err != nil {
                return
            }
        }

        for _, post := range res.Response.Posts {
            if post.id < sc.ids["lowest_id"] {
                sc.ids["lowest_id"] = post.id
            }

            if post.id > sc.ids["highest_id"] {
                sc.ids["highest_id"] = post.id
            }

            if post.id <= initHighestID {
                // probably wouldn't make it here??
                return
            }

            err = sc.scrapePost(post)
            if err != nil {
                return
            }
        }

        // i don't get this
        sc.offset += len(res.Response.Posts)
    }
}

func (sc *scrapeContext) scrapeBlog() (data *postsResponse, err error) {
    for data == nil {
        data, err = sc.scrapeBlogMaybe()
        if err != nil {
            return
        }
    }
    return
}

func (sc *scrapeContext) scrapeBlogMaybe() (*postsResponse, error) {
    // what the fuck
    sc.sema.Acquire(sc.offset)

    var (
        url *url.URL
        res *http.Response
        err error
    )

    // switch sc.state {
    // // ?????
    // case scrapeContextStateTryUseIndashAPI, scrapeContextStateUseIndashAPI:
    //     url = sc.getIndashBlogPoastURL()
    //     res, err = sc.doGetRequest(url, http.Header {
    //         "Referer": {"https://www.tumblr.com/dashboard"},
    //         "X-Requested-With": {"XMLHttpRequest"},
    //     })
    // default:
    //     url = sc.getAPIPostURL()
    //     res, err = sc.doGetRequest(url, nil)
    // }

    // the above switches between InDashAPI and the regular API
    url = sc.getAPIPostsURL()
    res, err = sc.doGetRequest(url, nil)

    if err != nil {
        return nil, err
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        // fuck the indash api
        // if sc.state = scrapeContextStateTryUseAPI && res.StatusCode == http.StatusNotFound && len(sc.scraper.cfg.Username) != 0 {
        //     sc.state = scrapeContextStateTryUseIndashAPI
        //     return nil, nil
        // }
        // if sc.state == scrapeContextStateTryUseIndashAPI && res.StatusCode != http.StatusNotFound {
        //     err := account.LoginOnce()
        //     if err != nil {
        //         return nil, err
        //     }
        //     sc.state = scrapeContextStateTryUseIndashAPI
        //     return nil, nil
        // }
        return nil, fmt.Errorf("GET %s failed with: %d %s", url, res.StatusCode, res.Status)
    }

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        return nil, err
    }

    data := &postsResponse{}
    err = json.Unmarshal(body, data)
    if err != nil {
        return nil, err
    }

    if sc.state == scrapeContextStateTryUseAPI {
        sc.state = scrapeContextStateUseAPI
    }

    return data, nil
}

func (sc *scrapeContext) scrapePost(post *post) error {
    // NPF post
    // is it worth going or not?
    err := sc.scrapeNPFContent(post, post.Content)
    if err != nil {
        return err
    }

    // actually get the content
    for _, t := range post.Trail {
        var cs []content
        err = json.Unmarshal(t.Content, &cs)
        if err != nil {
            continue
        }

        err = sc.scrapeNPFContent(post, cs)
        if err != nil {
            return err
        }
    }

    // indash posts
    // bodyScraped := false
    // for _, text := range []string{post.Body, post.Answer} {
    //     if len(text) != 0 {
    //         bodyScraped = true
    //         sc.scrapePostBody(post, text)
    //     }
    // }
    //
    // if !bodyScraped && len(post.Reblog.Comment) != 0 {
    //     sc.scrapePostBody(post, post.Reblog.Comment)
    // }
    //
    // for _, photo := range post.Photos {
    //     sc.downloadFileAsync(post, photo.OriginalSize.URL)
    // }
    //
    // if len(post.VideoURL) != 0 {
    //     sc.downloadFileAsync(post, post.VideoURL)
    // }

    return nil
}

func (sc *scrapeContext) scrapeNPFContent(post *post, cs []content) error {
    for _, c := range cs {
        // check if this post even has something to download
        if len(c.Media) == 0 {
            continue
        }

        switch c.Type {
        case "image":
            var ms imageMedia
            err := json.Unmarshal(c.Media, &ms)
            if err != nil {
                return err
            }

            bestURL := ms[0].URL
            bestArea := ms[0].Width * ms[0].Height

            for _, m := range ms {
                if m.HasOriginalDimensions {
                    bestURL = m.URL
                    break
                }
                if m.Width * m.Height > bestArea {
                    bestURL = m.URL
                }
            }

            sc.downloadFileAsync(post, bestURL)
        case "video":
            var ms videoMedia
            err := json.Unmarshal(c.Media, &ms)
            if err != nil {
                return err
            }

            if strings.Contains(ms.URL, "tumblr.com") {
                sc.downloadFileAsync(post, ms.URL)
            }
        }
    }

    return nil
}

// this is part of the InDashAPI
// func (sc *scrapeContext) scrapePostBody(post *post, text string) {
//     nodes, err := html.ParseFragement(strings.NewReader(text), &html.Node {
//         Type: html.ElementNode,
//         DataAtom: atom.Div,
//         Data: "div",
//     })
//
//     if err != nil {
//         log.Printf("%s: failed to parse body -> falling back to regexp: %v", sc.link, err)
//         sc.scrapePostBodyUsingSearch(post, text)
//         return
//     }
//
//     for len(nodes) != 0 {
//         idx := len(nodes) - 1
//
//         node := nodes[idx]
//         nodes[idx] = nil
//
//         nodes = nodes[0:idx]
//
//         for child := node.FirstChild; child != nil; child = child.NextSibling {
//             nodes = append(nodes, child)
//         }
//
//         if node.Type != html.ElementNode {
//             continue
//         }
//
//         for _, attr := range node.Attr {
//             switch attr.Key {
//             case "href", "src", "data-big-photo":
//                 if mediaURLRegexp.MatchString(attr.Val) {
//                     sc.downloadFileAsync(post, attr.Val)
//                 }
//             }
//         }
//     }
// }

// this is part of the InDashAPI
// func (sc *scrapeContext) scrapePostBodyUsingSearch(post *post, test string) {
//     for _, u := range htmlMediaURLRegexp.FindAllString(text, -1) {
//         sc.downloadFileAsync(post, u)
//     }
// }

func (sc *scrapeContext) downloadFileAsync(post *post, rawurl string) {
    if len(rawurl) == 0 {
        // lol how did we get here
        panic("missing url")
    }

    // wtf is this offset shit
    sc.sema.Acquire(sc.offset)
    sc.errgroup.Go(func() error {
        defer sc.sema.Release()
        return sc.downloadFile(post, rawurl)
    })
}

func (sc *scrapeContext) downloadFile(post *post, rawURL string) error {
    optimalRawURL := sc.fixupURL(rawURL)

    // first try to use the optimal URL, if that doesn't work then fall back on the original
    err := sc.downloadFileMaybe(post, optimalRawURL)
    if err == errFileNotFound && optimalRawURL != rawURL {
        err = sc.downloadFileMaybe(post, rawURL)
    }

    // ignore 404 errors
    if err == errFileNotFound {
        log.Printf("%s: did not find %s", sc.link, rawURL)
        err = nil
    }

    // not 100% sure when grabbing a file timesout, it cancells the context
    // should probably make ignore timeouts so that doesn't cancel the context
    if err != nil {
        log.Printf("%s: failed to download file: %v", sc.link, err)
    }

    return err
}

func (sc *scrapeContext) downloadFileMaybe(post *post, rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return err
    }

    path := filepath.Join(sc.config.Save, filepath.Base(rawURL))
    fileTime := post.timestamp() // im slightly confused about where this came from

    // file already exists -> skip
    _, err = os.Lstat(path)
    if err == nil {
        log.Printf("%s: skipping %s", sc.link, path)
        return nil
    }

    res, err := sc.doGetRequest(u, nil)
    if err != nil {
        return err
    }
    defer res.Body.Close()

    switch res.StatusCode {
    case http.StatusOK:
        // continue
    case http.StatusForbidden:
        // if a video got deleted for some reason, the link is 403 forbidden
        return nil
    case http.StatusNotFound:
        return errFileNotFound
    case http.StatusInternalServerError:
        return errFileNotFound
    default:
        return fmt.Errorf("GET %s failed with: %d %s", rawURL, res.StatusCode, res.Status)
    }

    lastModifiedString := res.Header.Get("Last-Modified")
    if len(lastModifiedString) != 0 {
        lastModified, err := time.Parse(time.RFC1123, lastModifiedString)
        if err != nil {
            log.Printf("%s: failed to parse Last-Modified header: %v", sc.link, err)
        } else if fileTime.Sub(lastModified) > 24*time.Hour {
            fileTime = lastModified
        }
    }

    fixedPath := sc.fixupFilePath(res, path)
    if fixedPath != path {
        path = fixedPath

        // file already exits -> skip
        _, err = os.Lstat(path)
        if err == nil {
            log.Printf("%s: skipping %s", sc.link, path)
            return nil
        }
    }

    if !acquireFile(path) {
        return nil
    }
    defer releaseFile(path)

    file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
    if err != nil {
        return nil
    }

    _, err = io.Copy(file, res.Body)
    if err != nil {
        _ = file.Close()
        _ = os.Remove(path)
        return err
    }

    err = file.Close()
    if err != nil {
        _ = os.Remove(path)
        return err
    }

    err = os.Chtimes(path, fileTime, fileTime)
    if err != nil {
        return err
    }

    log.Printf("%s: wrote %s", sc.link, path)
    return nil
}

func (sc *scrapeContext) getAPIPostsURL() *url.URL {
    u, err := url.Parse(fmt.Sprintf("https://api.tumblr.com/v2/blog/%s/posts", sc.link))
    if err != nil {
        panic(err)
    }

    vals := url.Values {
        "api_key": {sc.config.APIKey},
        "limit": {"20"},
        "npf": {"true"},
        "reblog_info": {"1"},
    }

    u.RawQuery = vals.Encode()

    return u
}

func (sc *scrapeContext) doGetRequest(url *url.URL, header http.Header) (*http.Response, error) {
    if header == nil {
        header = make(http.Header)
    }

    req := &http.Request {
        Method: http.MethodGet,
        URL: url,
        Header: header,
    }
    req = req.WithContext(sc.ctx)
    return sc.scraper.client.Do(req)
}

func (sc *scrapeContext) fixupURL(url string) string {
    if strings.HasSuffix(url, ".mp4") {
        return videoURLFixupRegexp.ReplaceAllString(url, ".mp4")
    }

    return imageSizeFixupRegexp.ReplaceAllString(url, "_1280.$1")
}

func (sc *scrapeContext) fixupFilePath(res *http.Response, path string) string {
    _, contentDispositionParams, _ := mime.ParseMediaType(res.Header.Get("Content-Disposition"))
    if contentDispositionParams != nil {
        filename := contentDispositionParams["filename"]
        if len(filename) != 0 {
            return filepath.Join(sc.config.Save, filename)
        }
    }

    exts, _ := mime.ExtensionsByType(res.Header.Get("Content-Type"))
    if len(exts) != 0 {
        dir, file := filepath.Split(path)
        curExt := filepath.Ext(file)

        // this seems pointless?
        for _, ext := range exts {
            if ext == curExt {
                return path
            }
        }

        basename := strings.TrimSuffix(file, curExt)
        file = basename + exts[0]
        return filepath.Join(dir, file)
    }

    return path
}
