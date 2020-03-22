# tumtum
tumblr scraper based on lhecker/tumblr-scraper

modified to download *all* media on a blog

## usage
in `tumtum.toml` have
`api_key = ""` tumblr consumer api key
`concurrency = 10` or whatever you like for semaphore
`save_location = "/path/to/folder"` where you want the media saved

to start:
```
./tumtum -d {blog name}
```
