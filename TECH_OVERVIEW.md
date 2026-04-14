# Technical Overview & Business Flow

A comprehensive reference for anyone new to this project. Covers all technologies, libraries, protocols, data flows, and system architecture.

---

## Table of Contents

1. [Technology Stack](#technology-stack)
2. [External Libraries & Dependencies](#external-libraries--dependencies)
3. [Internal Libraries](#internal-libraries)
4. [System Architecture](#system-architecture)
5. [Business Flows](#business-flows)
6. [Data Storage & Database](#data-storage--database)
7. [Security & Cryptography](#security--cryptography)
8. [Communication Protocols](#communication-protocols)
9. [Concurrency Model](#concurrency-model)
10. [VPN Protocols & Clients](#vpn-protocols--clients)
11. [Configuration & Deployment](#configuration--deployment)
12. [Testing](#testing)
13. [Glossary](#glossary)

---

## Technology Stack

| Layer | Technology | Version / Details |
|-------|-----------|-------------------|
| **Language** | Go | 1.24+ (with toolchain go1.24.1) |
| **Platform** | Telegram Bot API | Custom fork with reaction support |
| **Database** | BadgerDB v4 | Embedded key-value store with encryption |
| **Transport** | SSH | ED25519 key authentication |
| **VPN Protocols** | WireGuard, Outline (Shadowsocks), AmneziaVPN, Reality | Multiple protocol support |
| **Cryptography** | PBKDF2, HMAC-SHA256, AES (BadgerDB) | Key derivation and data encryption |
| **Architecture** | Monolithic, concurrent goroutines | Single binary, multi-goroutine |
| **OS Target** | Linux (primarily) | Designed for server deployment |
| **User Interface** | Telegram (Russian language) | Private chat + admin group chat |

---

## External Libraries & Dependencies

### Core Dependencies (Direct)

| Library | Import Path | Purpose | Where Used |
|---------|------------|---------|------------|
| **BadgerDB v4** | `github.com/dgraph-io/badger/v4` | Embedded, encrypted key-value database. Stores sessions, receipt queues, and photo hashes. Zero external DB dependency. | `main.go`, `session.go`, `rqueue.go`, `queue2.go`, `uniqphoto.go` |
| **Telegram Bot API** | `github.com/vpngen/embassy-tgbot/telegram-bot-api` (local fork) | Custom fork of go-telegram-bot-api with message reaction support for captcha system. | `bot.go`, `bot2.go`, `talks.go`, `talks2.go`, `ministry.go`, `rqueue.go`, `queue2.go`, `msg_sync.go`, `res.go`, `res2.go`, `reactions.go`, `download_*.go` |
| **VPNGen Ministry** | `github.com/vpngen/ministry` | Backend VPN management service types. Provides `Answer`, `VIPAnswer` response structs and API contracts. | `ministry.go`, `msg_sync.go`, `req_brigade.go` |
| **VPNGen Keydesk** | `github.com/vpngen/keydesk` | VPN key management models and utilities. Provides brigade configuration structures. | `ministry.go` |
| **VPNGen WordsGens** | `github.com/vpngen/wordsgens` | Random name and mnemonic word generation using Nobel Prize laureates (namesgenerator) and seed words (seedgenerator). | `ministry.go` |
| **Go SSH** | `golang.org/x/crypto/ssh` | SSH client for connecting to the Ministry backend server. | `ssh.go`, `ministry.go`, `msg_sync.go`, `stat_sync.go`, `req_brigade.go` |
| **WireGuard wgctrl** | `golang.zx2c4.com/wireguard/wgctrl` | WireGuard key generation (`wgtypes.GeneratePrivateKey`). Used in fake/test mode. | `ministry.go` |
| **Google UUID** | `github.com/google/uuid` | UUID generation for receipt queue IDs, session labels, and request tracking. | `session.go`, `rqueue.go`, `queue2.go`, `label.go`, `req_brigade.go` |
| **Transliterator** | `github.com/alexsergivan/transliterator` | Russian-to-Latin character transliteration for generating valid WireGuard config filenames. | `internal/kdlib/trans.go` |
| **btcutil/base58** | `github.com/btcsuite/btcd/btcutil` | Base58 encoding for certain identifiers. | `ministry.go` |
| **PBKDF2** | `golang.org/x/crypto/pbkdf2` | Password-Based Key Derivation Function for generating encryption keys from environment variable passphrases. | `config.go` |

### Indirect Dependencies (Transitive)

| Library | Purpose |
|---------|---------|
| `github.com/dgraph-io/ristretto/v2` | Cache layer used internally by BadgerDB |
| `github.com/dustin/go-humanize` | Human-readable formatting (BadgerDB internal) |
| `github.com/cespare/xxhash/v2` | Fast hashing (BadgerDB internal) |
| `github.com/klauspost/compress` | Compression (BadgerDB internal) |
| `github.com/google/flatbuffers` | Serialization (BadgerDB internal) |
| `golang.org/x/sync` | Synchronization primitives |
| `golang.org/x/text` | Text processing (transliteration support) |
| `golang.org/x/net` | Network utilities |
| `go.opencensus.io` | Distributed tracing (BadgerDB internal) |
| `github.com/go-openapi/*` | OpenAPI types (keydesk/ministry dependencies) |
| `github.com/vpngen/dc-mgmt` | Datacenter management types (ministry dependency) |
| `github.com/vpngen/vpngine` | VPN engine types (ministry dependency) |
| `github.com/SherClockHolmes/webpush-go` | Web push notifications (keydesk dependency) |
| `google.golang.org/protobuf` | Protocol Buffers (BadgerDB internal) |
| `go.mongodb.org/mongo-driver` | MongoDB driver types (strfmt dependency) |

---

## Internal Libraries

### `logs/` — Leveled Logging

**Package:** `logs`

A simple, custom logging library with 5 severity levels. Uses Go's `fmt.Fprintf` with atomic level checking. No external logging framework.

**Log Levels (ascending verbosity):**
```
0 = Critical  (always logged, stderr)
1 = Error     (stderr)
2 = Warning   (stderr)
3 = Info      (stdout)
4 = Debug     (stdout)
```

**API:** `logs.Debugf(format, args...)`, `logs.Infof(...)`, `logs.Warningf(...)`, `logs.Errf(...)`, `logs.Criticf(...)`

**Design Choice:** No log formatting, no structured logging, no file rotation. Output goes directly to stdout/stderr. Log rotation is expected to be handled by the deployment environment (systemd, Docker, etc.).

### `internal/kdlib/` — Keydesk Library

**Package:** `kdlib`

Network utilities and text processing:
- **`net.go`** — Random IP address generation within CIDR prefixes (IPv4 and IPv6). Used for generating fake VPN configurations during testing.
- **`trans.go`** — Russian-to-Latin transliteration for sanitizing file names. Ensures WireGuard `.conf` filenames only contain ASCII characters.

### `telegram-bot-api/` — Custom Telegram Bot API Fork

**Package:** `tgbotapi`

A customized fork of the popular `go-telegram-bot-api` library. Key additions:
- **Message Reaction support** (`MessageReactionUpdated`, `ReactionType`, `SetMessageReactionConfig`) — essential for the emoji-based captcha system.
- Standard Telegram Bot API features: sending messages, photos, documents, inline keyboards, callback queries, file downloads.

**Why a fork?** At the time of development, the upstream library did not support Telegram's reaction API, which is used for the bot's captcha verification system.

---

## System Architecture

### High-Level Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                          TELEGRAM CLOUD                           │
│                                                                   │
│   ┌─────────┐              ┌────────────┐                        │
│   │  Users  │◄────────────►│ Bot1 (API) │  Long Polling          │
│   └─────────┘              └─────┬──────┘                        │
│                                  │                               │
│   ┌─────────┐              ┌─────┴──────┐                        │
│   │ Admins  │◄────────────►│ Bot2 (API) │  Long Polling          │
│   └─────────┘              └─────┬──────┘                        │
│                                  │                               │
└──────────────────────────────────┼───────────────────────────────┘
                                   │
                     ┌─────────────┴─────────────┐
                     │    Embassy TGBot Process   │
                     │                            │
                     │  ┌──────────────────────┐  │
                     │  │     Goroutines       │  │
                     │  │                      │  │
                     │  │  • Bot1 Event Loop   │  │
                     │  │  • Bot2 Event Loop   │  │
                     │  │  • Queue1 Loop       │  │
                     │  │  • Queue2 Loop       │  │
                     │  │  • MsgSync Loop      │  │
                     │  │  • StatSync Loop     │  │
                     │  │  • Maintenance Loop  │  │
                     │  │  • BadgerDB GC       │  │
                     │  │  • Per-message       │  │
                     │  │    goroutines        │  │
                     │  └──────────┬───────────┘  │
                     │             │               │
                     │  ┌──────────┴───────────┐  │
                     │  │  BadgerDB (Encrypted)│  │
                     │  │  ├── Sessions        │  │
                     │  │  ├── Receipt Queue 1 │  │
                     │  │  ├── Receipt Queue 2 │  │
                     │  │  └── Photo Hashes    │  │
                     │  └──────────────────────┘  │
                     │             │               │
                     │  ┌──────────┴───────────┐  │
                     │  │  Label Storage (FS)  │  │
                     │  │  └── .log files      │  │
                     │  └──────────────────────┘  │
                     │             │               │
                     └─────────────┼───────────────┘
                                   │
                              SSH (port 22)
                              ED25519 auth
                                   │
                     ┌─────────────┴─────────────┐
                     │     Ministry Server        │
                     │  (Backend VPN Management)  │
                     │                            │
                     │  Commands:                 │
                     │  • createbrigade           │
                     │  • restorebrigadier        │
                     │  • reqvipid                │
                     │  • readmsgs                │
                     │  • sendlabels              │
                     │                            │
                     │  Manages:                  │
                     │  • VPN Server allocation   │
                     │  • Brigade lifecycle       │
                     │  • Key generation          │
                     │  • VIP processing          │
                     └────────────────────────────┘
```

### Goroutine Architecture

The application runs 8+ persistent goroutines plus per-message goroutines:

| Goroutine | Lifecycle | Poll Interval | Purpose |
|-----------|-----------|---------------|---------|
| `badgerGC` | Permanent | 5 minutes | Database value log garbage collection |
| `checkMantenance` | Permanent (if configured) | 15 seconds | Monitor maintenance state files |
| `runBot` | Permanent | Long-poll (3s timeout) | Process user Telegram updates |
| `runBot2` | Permanent | Long-poll (3s timeout) | Process admin Telegram updates |
| `ReceiptQueueLoop` | Permanent | 3 seconds | Process receipt queue 1 |
| `ReceiptQueueLoop2` | Permanent | 3 seconds | Process receipt queue 2 |
| `msgSyncLoop` | Permanent | 1 minute | Poll ministry for VIP messages |
| `statSyncLoop` | Permanent | 10 minutes | Send label logs to ministry |
| Per-message handlers | Ephemeral | N/A | One goroutine per incoming update |

---

## Business Flows

### Flow 1: New Brigade Registration (Standard User)

```
User                   Bot1                   BadgerDB          Queue1/2            Admin Group         Ministry
 │                      │                       │                  │                    │                  │
 │  /start             │                       │                  │                    │                  │
 │─────────────────────►│                       │                  │                    │                  │
 │                      │  Check/create session │                  │                    │                  │
 │                      │──────────────────────►│                  │                    │                  │
 │                      │                       │                  │                    │                  │
 │  Welcome + keyboard  │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │                      │                       │                  │                    │                  │
 │  Click "Хочу бригаду"│                       │                  │                    │                  │
 │─────────────────────►│                       │                  │                    │                  │
 │                      │  Captcha check        │                  │                    │                  │
 │  Emoji math puzzle   │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │                      │                       │                  │                    │                  │
 │  Set reaction (answer)                       │                  │                    │                  │
 │─────────────────────►│                       │                  │                    │                  │
 │                      │  Verify reaction      │                  │                    │                  │
 │  "Send any image"    │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │                      │                       │                  │                    │                  │
 │  [Photo attachment]  │                       │                  │                    │                  │
 │─────────────────────►│                       │                  │                    │                  │
 │                      │  PutReceipt           │                  │                    │                  │
 │                      │──────────────────────►│──────────────────►│                    │                  │
 │  "We're reviewing"   │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │                      │                       │                  │                    │                  │
 │                      │     Queue1 loop: download photo, hash, forward              │                  │
 │                      │                       │                  │  Photo + buttons   │                  │
 │                      │                       │                  │───────────────────►│                  │
 │                      │                       │                  │                    │                  │
 │                      │                       │                  │  Admin clicks      │                  │
 │                      │                       │                  │  "Подтвердить"     │                  │
 │                      │                       │                  │◄───────────────────│                  │
 │                      │                       │                  │                    │                  │
 │                      │     Queue2→Queue1 decision forwarding    │                    │                  │
 │                      │                       │◄─────────────────│                    │                  │
 │                      │                       │                  │                    │                  │
 │                      │     Queue1 loop: catchReviewedReceipt    │                    │                  │
 │                      │                       │                  │  SSH: createbrigade│                  │
 │                      │───────────────────────────────────────────────────────────────►│                  │
 │                      │                       │                  │                    │  VPN configs     │
 │                      │◄──────────────────────────────────────────────────────────────│                  │
 │                      │                       │                  │                    │                  │
 │  1. Brigadier name   │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │  2. Laureate bio     │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │  3. Seed explanation │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │  4. 6 magic words    │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │  5. Outline key      │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
 │  6. Activation link  │                       │                  │                    │                  │
 │◄─────────────────────│                       │                  │                    │                  │
```

### Flow 2: Brigade Recovery

```
User                   Bot1                   Ministry
 │                      │                       │
 │  Click "Восстановлю" │                       │
 │─────────────────────►│                       │
 │                      │                       │
 │  Enter brigadier name│                       │
 │─────────────────────►│                       │
 │                      │  Validate format      │
 │                      │  (adjective + surname) │
 │                      │                       │
 │  Enter 6 magic words │                       │
 │─────────────────────►│                       │
 │                      │  Generate name variants│
 │                      │  (ё↔е substitutions)   │
 │                      │  SSH: restorebrigadier │
 │                      │──────────────────────►│
 │                      │                       │
 │                      │  Brigade data         │
 │                      │◄──────────────────────│
 │                      │                       │
 │  Credentials (same   │                       │
 │  as new registration)│                       │
 │◄─────────────────────│                       │
```

### Flow 3: VIP Brigade Purchase

```
User                   Bot1                   Ministry           VIP Bot
 │                      │                       │                  │
 │  Click "VIP"         │                       │                  │
 │─────────────────────►│                       │                  │
 │                      │  SSH: reqvipid         │                  │
 │                      │──────────────────────►│                  │
 │                      │  Request UUID          │                  │
 │                      │◄──────────────────────│                  │
 │                      │                       │                  │
 │  Link to VIP bot     │                       │                  │
 │  with UUID param     │                       │                  │
 │◄─────────────────────│                       │                  │
 │                      │                       │                  │
 │  User opens VIP bot  │                       │                  │
 │─────────────────────────────────────────────────────────────────►│
 │  ... payment flow ... │                       │                  │
 │                      │                       │                  │
 │                      │                       │  VIP brigade     │
 │                      │                       │  created         │
 │                      │                       │                  │
 │                      │  msgSyncLoop polls     │                  │
 │                      │──────────────────────►│                  │
 │                      │  VIPAnswer data        │                  │
 │                      │◄──────────────────────│                  │
 │                      │                       │                  │
 │  VIP Credentials     │                       │                  │
 │◄─────────────────────│                       │                  │
```

### Flow 4: Receipt Review (Admin Side)

```
Admin                  Bot2                   Queue2             Queue1
 │                      │                       │                  │
 │  Photo appears in    │                       │                  │
 │  admin group with    │◄──────────────────────│◄─────────────────│
 │  decision buttons    │                       │                  │
 │                      │                       │                  │
 │  Click "👍 Подтвердить"                      │                  │
 │─────────────────────►│                       │                  │
 │                      │  UpdateReceipt2()     │                  │
 │                      │──────────────────────►│                  │
 │                      │                       │                  │
 │  Audit photo posted  │  Queue2Loop polls     │                  │
 │◄─────────────────────│  Finds decision       │                  │
 │                      │  Updates Queue1       │                  │
 │  Original msg deleted│──────────────────────────────────────────►│
 │◄─────────────────────│                       │                  │
 │                      │                       │  Queue1Loop      │
 │                      │                       │  picks up        │
 │                      │                       │  → creates brigade│
```

### Flow 5: Maintenance Mode

```
Filesystem             Maintenance Loop       Bot1/Bot2
 │                      │                       │
 │  .maintenance_full   │                       │
 │  file created        │                       │
 │──────────────────────►│                       │
 │                      │  Detect change         │
 │                      │  Report to admin chat  │
 │                      │──────────────────────►│
 │                      │                       │
 │                      │  All new requests      │
 │                      │  get maintenance msg   │
 │                      │                       │
 │  .maintenance_full   │                       │
 │  file deleted        │                       │
 │──────────────────────►│                       │
 │                      │  Back to normal        │
 │                      │──────────────────────►│
```

---

## Data Storage & Database

### BadgerDB

**Type:** Embedded, LSM-tree based key-value store  
**Encryption:** AES with key rotation (every 10 days)  
**Index Cache:** 100 MB  

**Data Collections:**

| Collection | Key Prefix | Key Structure | Value | TTL | Purpose |
|------------|-----------|---------------|-------|-----|---------|
| Sessions | `session_` | HMAC-SHA256(secret, chatID+"7=Swinee") | JSON `Session` | 30 days | User dialog state, captcha progress |
| Receipt Queue 1 | `receiptq1_` | Random UUID (16 bytes) | JSON `CkReceipt` | 48 hours | Photo submissions awaiting review |
| Receipt Queue 2 | `receiptq2_` | Random UUID (16 bytes) | JSON `CkReceipt2` | 48 hours | Admin review interface entries |
| Photo Hashes | `photo_` | SHA-256 of photo bytes | SHA-256 hash | 90 days | Duplicate photo detection |

**Garbage Collection:** Background goroutine runs value log GC periodically.

### File-Based Storage

| File | Format | Purpose |
|------|--------|---------|
| Label logs (`.log.current`) | `timestamp\|UUID\|label` per line | Active analytics tracking |
| Label logs (`.log`) | Same format | Completed logs ready for sync |
| Maintenance files | Plain text (message body) | `.maintenance_full`, `.maintenance_newreg` |
| SSH key | PEM (ED25519) | `id_ed25519` |
| Embedded image | PNG | `vgbs.png` (compiled into binary) |

---

## Security & Cryptography

### Key Derivation

```
Environment Variable (passphrase)
         │
         ▼
    PBKDF2 (SHA-256)
    Iterations: 4096
    Key Length: 32 bytes
    Salt: "we4;6prSfm_k+Gn" (default) or custom (format: "salt:key")
         │
         ▼
    Derived Key (used for DB encryption, session HMAC, queue HMAC)
```

### Session ID Derivation

```
Chat ID (int64) + Salt ("7=Swinee")
         │
         ▼
    HMAC-SHA256 (with session secret)
         │
         ▼
    Prefix with "session_"
         │
         ▼
    Database Key
```

### Photo Deduplication

```
Photo bytes (downloaded from Telegram CDN)
         │
         ▼
    SHA-256 hash
         │
    ┌────┴────┐
    │ Check   │ "photo_" + hash → exists?
    │ BadgerDB│
    └────┬────┘
         │
    Yes: Reject (duplicate)
    No:  Store hash (90-day TTL), proceed
```

### Telegram ID Obfuscation

For VIP features, Telegram chat IDs are XOR-masked before sending to the Ministry backend:
```
obfuscatedID = chatID ^ 24537551337805
```
This prevents the Ministry from knowing actual Telegram IDs.

### Message Protection

- Messages marked as `ProtectContent = true` prevent forwarding/copying
- Auto-delete recommendations (1-3 days)
- No PII collection (no phone, email, real name)

### Anti-Spam Measures

| Measure | Implementation |
|---------|---------------|
| Captcha | Emoji math puzzles with reaction-based answers |
| Escalating cooldowns | 1m → 3m → 5m → 10m → 15m → 30m between retries |
| Max captcha attempts | 5 per session |
| Photo uniqueness | SHA-256 hash stored for 90 days |
| Concurrency limiting | Max 1 active dialog per chat (ChatsWins) |
| Ban states | `SessionStatePayloadBan`, `SessionStateBanOnBan` |

---

## Communication Protocols

### Telegram Bot API (HTTPS Long Polling)

**Method:** Long polling via `getUpdates` endpoint  
**Timeout:** Configurable (default 3 seconds)  
**Update Types:** `message`, `callback_query`, `message_reaction`  
**Two separate bot instances:** Bot1 (users, private chats) and Bot2 (admins, group chat)  

### SSH to Ministry Backend

**Authentication:** ED25519 public key  
**Username:** `_kolyan_`  
**Timeout:** 5 seconds  
**Data Format:** HTTP chunked transfer encoding over stdout  

**Commands:**

| Command | Direction | Purpose |
|---------|-----------|---------|
| `createbrigade -ch -j -l <label> <token>` | Bot → Ministry | Create new VPN brigade |
| `restorebrigadier -ch -j <name> <words> <token>` | Bot → Ministry | Restore existing brigade |
| `reqvipid -ch -tgid <id> -l <label> ... <token>` | Bot → Ministry | Request VIP brigade UUID |
| `readmsgs -ch <token>` | Bot → Ministry | Poll for VIP messages |
| `sendlabels -ch <token>` | Bot → Ministry | Send analytics labels (stdin pipe) |

**Response Format:** HTTP chunked encoding parsed via `httputil.NewChunkedReader`, containing JSON payloads.

### HTTP (Photo Downloads)

**Usage:** Downloading user-submitted photos from Telegram's CDN  
**Method:** Standard `http.Get()` to Telegram file URL  

---

## Concurrency Model

### Synchronization Primitives

| Primitive | Usage |
|-----------|-------|
| `sync.WaitGroup` | Goroutine lifecycle management (graceful shutdown) |
| `sync.Mutex` | Protecting shared state (ChatsWins, LabelStorage, Maintenance) |
| `chan struct{}` | Stop signal broadcasting |
| `time.Timer` | Periodic polling in background goroutines |
| `atomic.Int32` | Log level (lock-free reads) |

### Per-Message Goroutines

Each incoming Telegram update spawns a new goroutine:
```go
go messageHandler(opts, update, ministry)
go buttonHandler(opts, update, ministry)  
go reactionHandler(opts, update)
go messageHandler2(opts, update)
go buttonHandler2(opts, update)
```

**Concurrency Safety:**
- `ChatsWins` mutex prevents multiple goroutines handling the same chat simultaneously
- BadgerDB provides ACID transactions for concurrent reads/writes
- `LabelStorage` mutex protects log file writes
- `Maintenance` mutex protects state reads/writes

### Graceful Shutdown

```go
kill := make(chan os.Signal, 1)
signal.Notify(kill, os.Interrupt, SIGTERM, SIGHUP, SIGINT, SIGQUIT)

<-kill          // Wait for shutdown signal
close(stop)     // Broadcast stop to all goroutines
waitGroup.Wait() // Wait for all goroutines to finish
```

---

## VPN Protocols & Clients

The system generates configurations for multiple VPN protocols:

| Protocol | Client App | Config Format | Status |
|----------|-----------|---------------|--------|
| **Outline (Shadowsocks)** | Outline Client | Access key string | Active (primary) |
| **AmneziaVPN** | AmneziaVPN | `.vpn` file | Active (conditional) |
| **WireGuard** | WireGuard | `.conf` file + QR code | Commented out (was active) |
| **IPSec/L2TP** | Native OS support | Manual settings (PSK, user, pass, server) | Commented out |
| **Reality (VLESS)** | Hiddify, v2ray | URI string | Active (conditional) |

### Supported Platforms (Download Links)

**Outline:** Android (Play Store), iOS (App Store), Windows, macOS, Linux, Chrome  
**AmneziaVPN:** Android (Play Store), iOS (App Store), Windows, macOS, Linux

---

## Configuration & Deployment

### Environment Variables

The application is configured entirely through environment variables (12-factor app):

```bash
# Required
EMBASSY_TOKEN=...           # Bot1 Telegram token
CHECKBOT_TOKEN=...          # Bot2 Telegram token
EMBASSY_BADGER_DIR=./db     # BadgerDB directory
EMBASSY_BADGER_KEY=...      # DB encryption passphrase
SESSION_SECRET=...          # Session HMAC secret
QUEUE_SECRET=...            # Queue HMAC secret
MINISTRY_IP=10.0.0.1        # Ministry server IP
MINISTRY_TOKEN=...          # Ministry auth token
SSHKEY_PATH=/keys           # Directory with id_ed25519

# Optional
SUPPORT_URL=https://t.me/support_bot
VIP_BOT_URL=https://t.me/vipgenbot
CHECK_BILL_CHAT=-123456789  # Admin group chat ID (negative for groups)
LABEL_FILENAME=/data/labels
MAINTENANCE_STATE_FILES_DIR=/data/maintenance
EMBASSY_UPDATE_TIMEOUT=3    # Telegram polling timeout
EMBASSY_DEBUG=4             # Log level (0-4)
BOT_DEBUG=1                 # Bot API debug (0/1)
```

### Testing Mode

Set `SSHKEY_PATH=FAKE` to enable fake mode:
- No SSH connections to Ministry
- Random brigade data generated locally
- Uses `namesgenerator`, `seedgenerator`, and `wgtypes` for fake data
- Suitable for development and UI testing

### Build

```bash
go build -o embassy-tgbot .
```

The binary embeds `vgbs.png` via `//go:embed`.

### Deployment Requirements

- Go 1.24+ (for building)
- Network access to Telegram API (api.telegram.org)
- SSH access to Ministry server (port 22)
- Writable filesystem for BadgerDB and label logs
- ED25519 SSH key pair (or FAKE mode)

---

## Testing

### Existing Tests

| File | Tests | Purpose |
|------|-------|---------|
| `captcha_test.go` | `TestGetCaptchaText` | Generates 100 captchas, validates output |
| `internal/kdlib/net_test.go` | Various | Tests IP address generation and manipulation |

### Running Tests

```bash
go test ./...
```

### Fake Mode for Integration Testing

With `SSHKEY_PATH=FAKE`, the entire user flow can be tested without a Ministry backend. The bot will generate random brigadier data and deliver it as if a real brigade was created.

---

## Glossary

| Term | Meaning |
|------|---------|
| **Brigade** | A VPN server unit managed by a brigadier with multiple user slots |
| **Brigadier** | A person who manages a brigade and distributes VPN keys to their network |
| **Ministry** | The backend server that manages VPN infrastructure, allocates servers, generates configs |
| **Keydesk** | A web-based dashboard (accessible only via VPN) for managing brigade members |
| **6 Magic Words** | A mnemonic seed phrase used with the brigadier name to recover brigade access |
| **Embassy** | This Telegram bot — the "diplomatic gateway" to VPN Generator |
| **Laureate** | Nobel Prize laureate — the naming convention for brigadiers (e.g., "Весёлый Эйнштейн") |
| **Receipt/Photo** | An image submitted by users as anti-spam verification |
| **Queue 1** | The user-side receipt queue (photo → admin review → brigade creation) |
| **Queue 2** | The admin-side review queue (photo displayed with decision buttons) |
| **VIP Brigade** | Premium brigade with higher limits, unique IP, and no deletion risk |
| **Maintenance Mode** | System state where new registrations are paused (full or newreg-only) |
| **Check Bot / Bot2** | The admin-facing Telegram bot used in the review group chat |
| **Label** | Analytics tracking identifier attached to user sessions |
| **Captcha** | Emoji-counting puzzle verified via Telegram message reactions |
| **Ecode** | Unique error tracking code generated per handler invocation |
| **Session Stage** | Current step in the dialog flow (0-8, covering main + restore tracks) |
| **Session State** | User status flags (none, active, banned, secondary, double-banned) |
| **BadgerDB** | Embedded Go key-value database with encryption support |
| **PBKDF2** | Password-Based Key Derivation Function 2 — used to derive encryption keys |
| **HMAC** | Hash-based Message Authentication Code — used for session key derivation |
| **Chunked Encoding** | HTTP transfer encoding used for SSH command responses |
