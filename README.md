<p align="center">
  <img src="https://i.imgur.com/QAlRAvC.png" alt="teleGofer Logo" width="320" />
</p>

<h1 align="center">TeleGofer</h1>

<p align="center">
  <b>A high-performance, full-featured Telegram MTProto 2.0 client framework written in pure Go.</b>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" /></a>
  <img src="https://img.shields.io/badge/status-active%20development-orange" alt="Status" />
  <img src="https://img.shields.io/badge/allocs%2Fop-0%20(hot%20path)-brightgreen" alt="Zero Allocs" />
</p>

---

teleGofer is an independently maintained, production-grade Telegram MTProto 2.0 client library written in idiomatic Go. Designed for broad Telegram API coverage alongside a specialized high-performance file transfer subsystem, teleGofer prioritizes bounded memory discipline, low CPU overhead, protocol correctness, and high sustained throughput.

## Key Features

- **Full MTProto 2.0 Scope**: Built for complete Telegram API coverage (authentication, users, chats, channels, messages, media, bots, updates, and events).
- **High-Performance Transfer Engine**: Engineered around streaming I/O, bounded concurrency, and chunked buffering (targeting 30+ MB/s sustained throughput without loading files into RAM).
- **Zero-Allocation TL Core**: Optimized Type Language (TL) binary encoder and decoder with zero allocations in critical serialization hot paths.
- **Strict Protocol Correctness**: Faithful adherence to official Telegram specifications (4-byte alignment, precise length prefixes, null padding verification).
- **Resilient Error Taxonomy**: Typed, unwrappable errors for `FLOOD_WAIT_X`, `FLOOD_PREMIUM_WAIT_X`, `*_MIGRATE_X`, and `FILE_REFERENCE_EXPIRED`.
- **Flexible Session Management**: Pluggable authorization persistence supporting in-memory and disk-backed storage with strict copy isolation.

---

## Requirements

- **Go 1.22+**
- Telegram `api_id` and `api_hash` from [my.telegram.org](https://my.telegram.org)

---

## Installation

```bash
go get github.com/mrabhi2k3/telegofer
```

---

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/mrabhi2k3/telegofer/client"
	"github.com/mrabhi2k3/telegofer/mtproto/session"
)

func main() {
	cfg := client.Config{
		APIID:   123456,
		APIHash: "0123456789abcdef0123456789abcdef",
		Session: session.NewFile("session.dat"),
	}

	tgClient, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Printf("teleGofer %s initialized successfully\n", client.Version)
	_ = tgClient
}
```

---

## Architecture Overview

teleGofer grows organically by subsystem—directories exist only when backed by functional, tested code:

```
teleGofer/
├── client/              # Public client API, configuration, logging
├── mtproto/             # MTProto 2.0 wire protocol
│   ├── auth/            # Diffie-Hellman key exchange, RSA encryption
│   ├── connection/      # Encrypted connection manager, keepalive, ACKs
│   ├── crypto/          # AES-256-IGE, KDF, factor, auth key
│   ├── protocol/        # Message framing, message ID, sequence generator
│   ├── rpc/             # Typed RPC errors, correlation engine, gzip, container
│   ├── session/         # Memory & file authorization session storage
│   └── transport/       # Abridged, Intermediate, Padded, Full TCP transports
├── tl/                  # Telegram Type Language core
│   ├── decoder/         # TL primitive deserialization (bounds-checked, zero-copy)
│   ├── encoder/         # TL primitive serialization (zero-alloc hot paths)
│   ├── generated/       # Generated MTProto and core Telegram API types
│   ├── generator/       # Go code generator from TL schema
│   └── parser/          # TL schema parser and lexer
├── dc/                  # Data Center endpoints and automatic migration manager
└── transfer/            # High-speed streaming upload & download engine
```

---

## License

Distributed under the [MIT License](LICENSE).
