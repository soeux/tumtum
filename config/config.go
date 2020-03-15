package config

import (
    "os"
    "github.com/pelletier/go-toml"
)

// debating if i should just make these cli arguments or just have it save in here
type Config struct {
    APIKey string `toml:"api_key"`
    Concurrency int `toml:"concurrency"`
    Save string `toml:"save_location"`
}

func LoadConfigOrDefault(path string) (*Config, error) {
    cfg, err := loadConfig(path)
    if err != nil {
        if !os.IsNotExist(err) {
            return nil, err
        }

        cfg, err = loadConfig(path + ".bak")
        if err != nil {
            if !os.IsNotExist(err) {
                return nil, err
            }

            log.Print("config file not found -> using default values [exit]")
        } else {
            log.Print("recovering backup config file")
        }
    }

    // i have no idea where this is from
    if cfg.Concurrency <= 0 {
        cfg.Concurrency = 24
    }

    return cfg, nil
}

func loadConfig(path string) (*Config, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    cfg := &Config{}

    err = toml.NewDecoder(f).Decode(cfg)
    if err != nil {
        return nil, err
    }

    return cfg, nil
}
