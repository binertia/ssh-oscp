# ssh-art

> a living Cybersec relax quiz SSH-based terminal to make you relax.


I just stress to much on that test. so i just want to make something for relax myself.
SSH into a shared terminal space where neo rain falls, stars, gargantua , while you fight lovely OSCP relax quiz


---

Each room features:
- 🎨 ANSI animation (12 FPS, ~24KB/s per client)
- 📝 first room collaborative wall text (incase someone want to tell his server domain)
- 👥 live multiplayer (let's pass that test)

---

## 🚀 Quick Start

### Prerequisites

- [Go](https://go.dev/) 1.26+
- `ssh-keygen` (for host key generation, or provide your own)

### Run locally

```bash
git clone <repo-url> <dir>
cd <dir>
go run .
```

then connect:

```bash
ssh localhost -p 2222
```

no password required—this is public art, not a vault.

### Build

```bash
go build -o ssh-art .
./ssh-art
```

---

## 🎮 Controls

| Key | Action |
|-----|--------|
| `1` | enter **neoRain** |
| `2` | enter **start shooting** |
| `3` | enter **gargantua** |
| `4` | enter **4D terminal** |
| `5` | enter **don't give up robot** |
| `r` | reset quiz / next quiz (1 sec cd tho) |
| `DEL` / `BS` | del strings other write |

---

## 🏛️ Architecture

```
┌─────────────────────────────────────────────┐
│            SSH Museum Server                │
│                                             │
│  ┌─────────┐   ┌──────────┐   ┌─────────┐   │
│  │  Wish   │◄─►│  Room    │◄─►│ Session │   │
│  │ Server  │   │ Engine   │   │ Writer  │   │
│  └─────────┘   │(1 goroutine  │         │   │
│                │ per room)│   │         │   │
│                └────┬─────┘   └─────────┘   │
│                     │                       │
│                ┌────┴────┐                  │
│                │  Frame  │                  │
│                │ Channel │                  │
│                └─────────┘                  │
└─────────────────────────────────────────────┘
```

- **One goroutine per room** 
- **Channels over locks**
- **ANSI-first rendering**
- **Flat packages** — `main` → `internal/rooms` → (`internal/ansi`, `internal/memory`)

---

## 🛤️ Roadmap

WIP :: after pass OSCP maybe

---

## 🎨 Design Principles

- **terminal itself is the canvas**

---

## 📜 License

MIT

---

> *"try harder, together"*
