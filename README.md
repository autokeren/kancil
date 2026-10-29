# Kancil 🦌

**The clever agent that lives in your Messenger.**

Send a text, a voice note, or a photo — Kancil builds it: websites that go live instantly, images that arrive as native picture messages, long-form content — all inside the chat you already use every day.

Built entirely on the **Meta Model API** and the **Messenger Platform** for the Meta Global AI Developer Hackathon (Oct 27 – Nov 4, 2026).

---

## Why Kancil exists

The agentic wave (autonomous coding agents, agent harnesses) lives on **Telegram and WhatsApp**. **Messenger — a billion-user channel and the default messaging surface across Southeast Asia, including Indonesia where this was built — has no serious agent harness.** Everyone follows the easy deploy path; the big channel stayed empty.

Kancil fills it: **an agent harness that lives natively in Messenger**, where UMKM owners, students, and warung operators already are. They will never install a CLI, never open a terminal — but they voice-note their friends all day. Kancil meets them there.

> Kancil doesn't fight the big animals — it outsmarts them. 🦌

---

## What it does (all verified live)

| You send in Messenger | Kancil does | You receive back |
|---|---|---|
| **"bikin website kedai kopi Kopi Sore"** | Muse Spark plans → writes `index.html` → publishes the site | Live public URL + the site, instantly |
| **"bikin gambar kucing oren pakai topi wizard"** | Muse Image generates the image | The image, **as a native picture message in the chat** |
| **🎙 a voice note** (Bahasa Indonesia or English) | Muse Voice Transcribe → agent loop | Text reply, built task — voice notes are first-class input |
| **📷 a photo** (e.g. a menu, a shop sign) | Muse Spark reads the image (multimodal input) | A site built from *what's in your photo* |
| **"ganti judul jadi X"** | `edit_file` — one targeted string replace | Refined site, no full rewrite |
| Any build request | Ghostfox MCP: agent **opens the live site in its stealth browser, takes a screenshot, and verifies its own work** | **Screenshot of the result, sent to your chat** |

### The self-verification loop

Most bots just answer. Kancil closes the loop autonomously:

```
"bikin website apotik sehat"
  → write_file  (builds index.html)
  → publish_site (live URL)
  → [MCP: ghostfox] page_open (opens the live site)
  → [MCP: ghostfox] page_screenshot (verifies it rendered)
  → send_photo (screenshot lands in your Messenger)
  → "Udah jadi — ini tampilannya, gua cek sendiri 🦌📸"
```

An agent that builds, **sees**, and proves its work — inside a chat app.

---

## The all-Meta stack

| Layer | Meta product |
|---|---|
| Brain | **Muse Spark 1.3** — native tool calling + MCP support (1M context) |
| Channel | **Messenger Platform** (webhooks, Send API, sender actions, attachments) |
| Voice | **Muse Voice Transcribe 1.0** |
| Images | **Muse Image 1.0** |
| Multimodal sight | Muse Spark reads photos sent in the chat |
| Credits | $150 Meta Model API credits (hackathon allocation) |

Plus our own **Ghostfox MCP** stealth browser engine (patched-Firefox) as the agent's eyes for self-verification — connected through the **MCP bridge**, the same protocol Muse Spark 1.3 advertises native support for.

---

## Architecture

```
   Messenger user (phone)
        │ voice note / photo / text
        ▼
   Meta Messenger Platform (webhook)
        ▼
┌──────────────────────────────────────────┐
│ KANCIL — Go agent harness                │
│                                          │
│  ┌───────────── agent loop ────────────┐ │
│  │ Muse Spark 1.3 (reasoning brain)    │ │
│  │        │ tool calls                 │ │
│  │        ▼                            │ │
│  │  ┌─ native tools ────────────────┐  │ │
│  │  │ write_file · edit_file        │  │ │
│  │  │ read_file · list_files        │  │ │
│  │  │ gen_image (Muse Image)        │  │ │
│  │  │ publish_site (static host)    │  │ │
│  │  │ send_photo (chat attachment) │  │ │
│  │  └──────────────────────────────┘  │ │
│  │  ┌─ MCP tools (bridge) ─────────┐  │ │
│  │  │ ghostfox: page_open,         │  │ │
│  │  │ page_screenshot, …(72 tools) │  │ │
│  │  │ + any stdio MCP server       │  │ │
│  │  └──────────────────────────────┘  │ │
│  │  memory: per-user JSON,            │ │
│  │  85% context compaction (opencode   │ │
│  │  style — Muse summarizes old turns) │ │
│  └────────────────────────────────────┘ │
│  voice → Muse Voice Transcribe           │
│  photos → multimodal content parts      │
│  typing indicator (auto-refresh 12s)     │
│  ack: model-decided, model-written      │
└──────────────────────────────────────────┘
        │ replies + images + site links
        ▼
   back in the chat — as native messages
```

### Design notes

- **Zero hardcoded replies.** Every user-facing sentence — including the quick *"Siap, gua kerjain ya!"* acknowledgment — is written by Muse in the user's language. The ack is fired by a parallel intent-router request to the model; if the message is just a question, Muse replies `SKIP` and no ack is sent.
- **Memory that survives restarts.** Per-user conversation history persists as JSON and reloads on boot. When prompt tokens pass 85% of the 1M context window, Muse summarizes older turns — recent context stays verbatim.
- **Multimodal input.** Voice notes arrive as audio attachments → downloaded → Muse Voice Transcribe → the transcript enters the agent loop flagged as a voice note, so the agent can reference how the user spoke. Photos arrive as image attachments → base64 → multimodal message parts — Muse Spark reads them natively.
- **MCP bridge.** Any stdio MCP server listed in the config attaches at runtime; its tools (with full JSON schemas) merge into the model's toolset. Ghostfox brings 72 browser tools. The bridge is the same protocol Muse Spark 1.3 ships native support for.

---

## Quick start

```bash
# (npm package coming; for now build from source)
go build -o kancil .

# 3-step wizard: Meta API key (verified live) → Messenger tokens → workspace
./kancil

# try it locally first
./kancil chat
bikin website kedai kopi Kopi Sore

# go live: webhooks + static site server
./kancil serve
```

Config lives in `~/.kancil/config.yaml` (see `config.example.yaml` — never commit your real one).

**Bot UX niceties:** instant typing indicator (auto-refreshed every 12s while Muse reasons) — and the quick *"Siap, gua kerjain ya 🦌"* acknowledgment is written by Muse itself, in your language, decided by a parallel intent-router call. Zero hardcoded replies.

---

## What the judges can verify

- **Public repo** — this repository (MIT license)
- **Working demo app** — Kancil runs on the author's Facebook Page (`autokeren.com`) in Messenger **development mode**; the demo video shows the complete loop on a real phone
- **Meta products used** — Muse Spark 1.3 (brain, reasoning + tool calling), Messenger Platform (channel, webhooks, sender actions, Send API with native image attachments), Muse Image 1.0, Muse Voice Transcribe 1.0, multimodal photo understanding — all through the Meta Model API
- **MCP** — the same protocol Muse Spark advertises native support for; Ghostfox MCP server (our open-source stealth browser) attaches 72 browser tools at runtime

> **Demo runs in Messenger app development mode** (standard pre-launch). Production rollout requires Meta's standard app review — the harness is production-ready beyond that gate.

---

## Limitations, honestly

- **No audio replies yet** — the Meta Model API catalog has speech-to-text (Muse Voice Transcribe) but no text-to-speech model, so Kancil replies in text (plus images). All-Meta, no third-party TTS bolted on. Roadmap: voice replies when Meta ships TTS.
- **Single-agent-per-user** — each user gets their own Kancil session and memory; there's no team/collaboration layer yet.
- **Dev-mode Messenger** — only app-role accounts (admin/developer/tester) can talk to the agent today; app review is the production gate.

## Roadmap

- WhatsApp Business channel (Meta's own WhatsApp Business Tools MCP) — the harness is channel-agnostic; the second Meta surface is a config entry away
- SQLite-backed long-term memory (cross-session recall: "remember that my shop is called Kopi Sore")
- Production Messenger app review
- More MCP servers — any stdio MCP server becomes agent tooling via config

## Credits

Built by **Ajat Sudrajat** ([github.com/autokeren](https://github.com/autokeren)) in 4 days during the Meta Global AI Developer Hackathon. The agent-loop patterns build on the author's open-source **Gulutux agent engine** (github.com/autokeren/gulutux) — for this hackathon the Meta integration (provider, channels, MCP bridge, memory, multimodal flows) was built from scratch.

MIT License. 🦌
