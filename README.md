# tumtum
tumblr scraper based on [lhecker/tumblr-scraper](https://github.com/lhecker/tumblr-scraper)

modified to download *all* media on a blog

## usage
in `tumtum.toml` you will need to specify a tumblr api key and a folder where you want everything saved to. concurrency is up to you.
```toml
api_key = ""
concurrency = 10
save_location = "/path/to/folder"
```


to start:
```
./tumtum -d {blog name}
```

## note
please note that i only did this to fulfill a specific purpose of grabbing everything off of one particular blog. if you have multiple blogs you want to scrape you will have to delete `tumtum.db` in order for the scraper to start on `time.Now()`. or take the code and do what you will.