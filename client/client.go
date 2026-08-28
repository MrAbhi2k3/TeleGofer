package client

import (
	"context"
	"fmt"
	"time"

	"github.com/mrabhi2k3/telegofer/mtproto/auth"
	"github.com/mrabhi2k3/telegofer/mtproto/connection"
	"github.com/mrabhi2k3/telegofer/mtproto/transport"
	"github.com/mrabhi2k3/telegofer/tl"
	"github.com/mrabhi2k3/telegofer/transfer"
)

const Version = "0.1.0-dev"

type Client struct {
	cfg  Config
	log  Logger
	conn *connection.Connection
}

func NewClient(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.applyDefaults()

	return &Client{
		cfg: cfg,
		log: cfg.Logger,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	c.log.Info("Connecting to Telegram...", "addr", c.cfg.ServerAddr)

	connectCtx, cancel := context.WithTimeout(ctx, c.cfg.ConnectTimeout)
	defer cancel()

	tr, err := transport.Dial(connectCtx, "tcp", c.cfg.ServerAddr, transport.NewIntermediate())
	if err != nil {
		return fmt.Errorf("client: dial failed: %w", err)
	}

	authKey, salt, err := auth.CreateAuthKey(connectCtx, tr, 2)
	if err != nil {
		tr.Close()
		return fmt.Errorf("client: auth failed: %w", err)
	}

	conn, err := connection.NewConnection(tr, authKey, salt)
	if err != nil {
		tr.Close()
		return fmt.Errorf("client: conn init failed: %w", err)
	}

	c.conn = conn
	c.log.Info("Connected to Telegram")
	return nil
}

func (c *Client) Invoke(ctx context.Context, req tl.Object) ([]byte, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("client: not connected")
	}
	timeout := c.cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return c.conn.Invoke(ctx, req, timeout)
}

func (c *Client) Uploader() *transfer.Uploader {
	return transfer.NewUploader(c.conn, c.cfg.UploadWorkers, c.cfg.ChunkSize)
}

func (c *Client) Downloader() *transfer.Downloader {
	return transfer.NewDownloader(c.conn, c.cfg.DownloadWorkers, c.cfg.ChunkSize)
}

func (c *Client) Close() error {
	c.log.Info("Closing Telegram client")
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

