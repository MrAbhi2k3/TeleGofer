package client

import (
	"fmt"
	"time"

	"github.com/mrabhi2k3/telegofer/mtproto/session"
)

type Config struct {
	APIID   int
	APIHash string

	ServerAddr string

	Session session.Storage
	Logger  Logger

	DeviceModel   string
	SystemVersion string
	AppVersion    string
	LangCode      string

	MaxRetries     int
	ConnectTimeout time.Duration
	RequestTimeout time.Duration

	DownloadWorkers int
	UploadWorkers   int
	ChunkSize       int
}

func (c *Config) applyDefaults() {
	if c.ServerAddr == "" {
		c.ServerAddr = "149.154.167.51:443"
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 5
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 30 * time.Second
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = 60 * time.Second
	}
	if c.DownloadWorkers == 0 {
		c.DownloadWorkers = 4
	}
	if c.UploadWorkers == 0 {
		c.UploadWorkers = 4
	}
	if c.ChunkSize == 0 {
		c.ChunkSize = 512 * 1024
	}
	if c.DeviceModel == "" {
		c.DeviceModel = "teleGofer"
	}
	if c.SystemVersion == "" {
		c.SystemVersion = "1.0"
	}
	if c.AppVersion == "" {
		c.AppVersion = Version
	}
	if c.LangCode == "" {
		c.LangCode = "en"
	}
	if c.Logger == nil {
		c.Logger = NopLogger{}
	}
	if c.Session == nil {
		c.Session = &session.Memory{}
	}
}

func (c *Config) validate() error {
	if c.APIID == 0 {
		return fmt.Errorf("telegofer: API ID is required")
	}
	if c.APIHash == "" {
		return fmt.Errorf("telegofer: API hash is required")
	}
	if c.ChunkSize != 0 {
		if c.ChunkSize%1024 != 0 {
			return fmt.Errorf("telegofer: chunk size must be divisible by 1024")
		}
		if (512*1024)%c.ChunkSize != 0 {
			return fmt.Errorf("telegofer: 512KB must be evenly divisible by chunk size")
		}
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("telegofer: max retries must be non-negative")
	}
	return nil
}

