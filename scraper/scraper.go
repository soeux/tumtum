package scraper

import (
    "os"
    "io"
    "io/ioutil"
    "fmt"
    "log"
    "net/url"
    "net/http"
    "math"
    "mime"
    "time"
    "regexp"
    "strings"
    "errors"
    "context"
    "encoding/json"
    "path/filepath"
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

// scraper object
type Scraper struct {
    client *http.Client
    config *config.Config
    db *database.Database
}

// initalising a scraper obj
func NewScraper(client *http.Client, database *database.Database) *Scraper {
    return &Scraper {
        client: client,
        databse: database,
    }
}

// creating the save location + starting a child process for scraper
<<<<<<< HEAD
func (s *Scraper) Scrape(ctx context.Context, link string, cfg *config.Config,) (time.Time, error) {
    err := os.MkdirAll(cfg.Save, 0755)
    if err != nil {
        return time.Time(), err
=======
func (s *Scraper) Scrape(ctx context.Context, link string, cfg *config.Config) (time.Time, error) {
    err := os.MkdirAll(cfg.Save, 0755)
    if err != nil {
        return time.Now(), err
>>>>>>> ecff6d204e6c32c29629e333251664dca54704f4
    }

    eg, ctx := errgroup.WithContext(ctx)

    sc := newScrapeContext(s, cfg, link, eg, ctx)
    if err != nil {
<<<<<<< HEAD
        return time.Time(), err
=======
        return time.Now(), err
>>>>>>> ecff6d204e6c32c29629e333251664dca54704f4
    }

    err = sc.Scrape()
    if err != nil {
<<<<<<< HEAD
        return time.Time(), err
=======
        return time.Now(), err
>>>>>>> ecff6d204e6c32c29629e333251664dca54704f4
    }

    return sc.time_obj, nil
}

type scrapeContext struct {
    // structuralised arguments
    scraper *Scraper
    config *config.Config
    link string
    errgroup *errgroup.Group
    ctx context.Context

    // current pagination state
<<<<<<< HEAD
	time_obj time.Time
	time_new bool
=======
	// TODO: this needs to be looked at more closely
    offset int
    before time.Time
	time_obj time.Time

    // informational values
    // highest_id, lowest_id, current_id
    ids map[string]int64
>>>>>>> ecff6d204e6c32c29629e333251664dca54704f4

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
		time_obj: time.Time(), // if this is left alone, the scraper will not work
		was_new: false,
        sema: semaphore.NewPrioritySemaphore(s.config.Concurrency),
    }

	if t, err := s.db.GetTime(link); err != nil {
		// time.Time() -> 0001-01-01 00:00:00 +0000 UTC
		// if there's no time then the time is now
		if t == time.Time() {
			// starting from the top
			sc.time_obj = time.Now()
			sc.time_new = true
		} else {
			// if there's time in the db, then we'll pick up where we left off
			sc.time_obj = t
		}
	} else {
		log.Printf("loading time from db failed")
	}

    return sc
}

func (sc *scrapeContext) Scrape() (err error) {
    log.Printf("%s: scraping starting at %v", sc.link, sc.time_obj)
    defer func() { log.Printf("%s: scraping finished at %v", sc.link, sc.time_obj) }()

    defer func() {
        e := sc.errgroup.Wait()
        if err != nil {
            err = e
        }
    }()

    startTime := sc.time_obj

    for {
		log.Printf("%s: fetching posts before %v", sc.link, sc.time_obj.Format("2Jan06 15:04:05"))

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
		// not sure if this going to be useful
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

    // the above switches between InDashAPI and the regular API
    url = sc.getAPIPostsURL()
    res, err = sc.doGetRequest(url, nil)

    if err != nil {
        return nil, err
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
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
