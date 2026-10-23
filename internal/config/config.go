package config

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig  `yaml:"server"`
	Routes []RouteConfig `yaml:"routes"`
}

type ServerConfig struct {
	ListenAddr      string        `yaml:"listen_addr"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type RouteConfig struct {
	Path         string        `yaml:"path"`
	MatchType    string        `yaml:"match_type"`
	Methods      []string      `yaml:"methods"`
	RateLimitRPS int           `yaml:"rate_limit_rps"`
	Backend      BackendConfig `yaml:"backend"`
	Mirror       *MirrorConfig `yaml:"mirror,omitempty"`
}

type BackendConfig struct {
	URL         string `yaml:"url"`
	FallbackURL string `yaml:"fallback_url"`
}

type MirrorConfig struct {
	URL        string `yaml:"url"`
	Percentage int    `yaml:"percentage"`
}

type Manager struct {
	configPath string
	current    atomic.Value
}

func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
	}
}

func (m *Manager) Load() error {
	file, err := os.Open(m.configPath)
	if err != nil {
		fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)

	if err := decoder.Decode(&cfg); err != nil {
		fmt.Errorf("decode config: %w", err)
	}

	if err := m.validate(&cfg); err != nil {
		fmt.Errorf("validate config: %w", err)
	}

	m.current.Store(&cfg)

	return nil
}

func (m *Manager) Get() *Config {
	return m.current.Load().(*Config)
}

func (m *Manager) validate(cfg *Config) error {
	for _, route := range cfg.Routes {
		if route.Mirror != nil {
			if route.Mirror.Percentage < 0 || route.Mirror.Percentage > 100 {
				return &yaml.TypeError{
					Errors: []string{"mirror percentage must be between 0 and 100"},
				}
			}
		}

		if route.MatchType != "exact" && route.MatchType != "prefix" && route.MatchType != "regex" {
			return &yaml.TypeError{
				Errors: []string{"invalid match_type: must be exact, prefix or regex"},
			}
		}
	}
	return nil
}

func (m *Manager) Watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("new watcher: %w", err)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					log.Println("[Config Manager] Configuration file modified. Reloading...")
					if err := m.Load(); err != nil {
						log.Printf("[Config Manager Error] Failed to reload config: %v. Using previous config.", err)
					} else {
						log.Println("[Config Manager] Configuration successfully reloaded!")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[Config Manager Error] Watcher error: %v", err)
			}
		}
	}()

	return watcher.Add(m.configPath)
}
