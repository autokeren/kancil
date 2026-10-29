# Kancil 🦌

**The clever agent that lives in your Messenger.**

Send a text, a voice note, a photo, or a video — Kancil builds it: websites that go live instantly, images that arrive as native picture messages, long-form content — all inside the chat you already use every day.

Built entirely on the **Meta Model API** and the **Messenger Platform** for the Meta Global AI Developer Hackathon (Oct 27 – Nov 4, 2026).

---

## Why Messenger — the market case

### The gap is real

Every serious agent harness (OpenClaw, autonomous coding CLIs, AI assistant frameworks) targets **Telegram or WhatsApp**. **Messenger has zero.** Not because it's hard — because Telegram bots deploy in 5 minutes while Messenger requires Meta app review. Everyone follows the easy path; the billion-user channel stayed empty.

Kancil fills it.

### The numbers

| | |
|---|---|
| **Messenger MAU** | 1+ billion globally (Meta) |
| **Indonesia Facebook users** | 140M+ — one of the largest markets in the world |
| **Indonesian UMKM (small businesses)** | 64+ million enterprises, most use Facebook Pages |
| **Voice note usage in Indonesia** | Dominant — people voice-note more than they type |
| **Agent-as-a-Service market** | $47B projected by 2030 (Grand View Research) |
| **Current competitors on Messenger** | Only scripted bots (ManyChat) — no autonomous agent harness |

### Who Kancil serves — "Build for Your World"

This is built for **Indonesian UMKM owners, students, and warung operators** who will never install a CLI or open a terminal. They already live in Messenger every day. They voice-note their friends all day. Kancil meets them exactly where they are:

- **Warung owner** voice-notes: *"bikin website kedai kopi gua"* → live website in the chat
- **Student** sends a photo of their notes → agent builds a study site from it
- **UMKM seller** sends a video of their product → agent describes it and creates a landing page

No new app to install. No terminal to learn. Just chat — in **Bahasa Indonesia** — with an agent that actually builds things.

### Why now

1. **Muse Spark 1.3 ships native tool calling + MCP support** — the agentic capabilities finally exist on Meta's own models
2. **Messenger Platform APIs are mature** — webhooks, sender actions, native image attachments, voice/audio handling are all stable and well-documented
3. **The agentic wave has proven the model** (Telegram/WA harnesses have traction) but **nobody brought it to the biggest channel in Southeast Asia**
4. **Indonesia's digital economy** is a government priority — UMKM digitalization is a national agenda

> Kancil doesn't fight the big animals — it outsmarts them. 🦌

---

## What it does (all verified live)

| You send in Messenger | Kancil does | You receive back |
|---|---|---|
| **"bikin website kedai kopi Kopi Sore"** *(Indonesian: "build me a coffee shop website")* | Muse Spark plans → writes `index.html` → publishes the site | Live public URL + the site, instantly |
| **"bikin gambar kucing oren pakai topi wizard"** *(Indonesian: "make me an image of an orange cat wearing a wizard hat")* | Muse Image generates the image | The image, **as a native picture message in the chat** |
| **🎙 a voice note** (Bahasa Indonesia or English) | Muse Voice Transcribe → agent loop | Text reply, built task — voice notes are first-class input |
| **📷 a photo** (e.g. a menu, a shop sign) | Muse Spark reads the image (multimodal input) | A site built from *what's in your photo* |
| **🎥 a video** (from your gallery or camera) | Muse Spark reads the visual sequence AND embedded audio together — one upload, both description and transcript | What happened in the video, described in text |
| **🎙 "buka detik.com, apa berita utamanya?"** | Agent opens Ghostfox browser → reads the page → summarizes headlines | Live web content in your chat — from a voice note |
| **"ya coba buka"** (follow-up) | Agent remembers context from previous message → opens Google News instead | Contextual memory — it understands what you mean |
| **"ganti judul jadi X"** *(Indonesian: "change the title to X")* | `edit_file` — one targeted string replace | Refined site, no full rewrite |
| Any build request | Ghostfox MCP: agent **opens the live site in its stealth browser, takes a screenshot, and verifies its own work** | **Screenshot of the result, sent to your chat** |

### The self-verification loop

Most bots just answer. Kancil closes the loop autonomously:

```
"buka detik.com dan ceritain berita utamanya" *(voice note)* *(Indonesian)*
  → write_file  (builds index.html)
  → publish_site (live URL)
  → [MCP: ghostfox] page_open (opens the live site)
  → [MCP: ghostfox] page_screenshot (verifies it rendered)
  → send_photo (screenshot lands in your Messenger)
  → "Udah jadi — ini tampilannya, gua cek sendiri 🦌📸"
     *(Indonesian: "Done — here's what it looks like, I checked it myself")*
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
| Multimodal video | Muse Spark reads video (visual + embedded audio) via `video_url` content parts |
| Speech to text | Muse Voice Transcribe 1.0 — official `/v1/asr/transcribe` endpoint, 24kHz mono PCM WAV |
| Multilingual voice | Voice notes work in Bahasa Indonesia AND English — the agent responds in the language spoken |
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

- **Real-time progress updates.** During multi-step builds, the agent sends tool progress messages to the chat: *"📝 Nulis kode… → 🚀 Publish website… → 👀 Buka website… → 📸 Screenshot…"* — the user watches the agent work in real time, never staring at a silent typing indicator. These are system progress indicators (like terminal output), not AI replies.
- **Browser session reuse.** Ghostfox browser sessions persist across messages — the agent can open a website, then open another in the same session, maintaining state.
- **Zero hardcoded replies.** Every user-facing sentence — including the quick *"Siap, gua kerjain ya!"* (Indonesian: "On it, I'll build that for you!") acknowledgment — is written by Muse in the user's own language. The ack is fired by a parallel intent-router request to the model; if the message is just a question, Muse replies `SKIP` and no ack is sent.
- **Memory that survives restarts.** Per-user conversation history persists as JSON and reloads on boot. When prompt tokens pass 85% of the 1M context window, Muse summarizes older turns — recent context stays verbatim.
- **Multimodal input.** Voice notes arrive as audio attachments → downloaded → Muse Voice Transcribe → the transcript enters the agent loop flagged as a voice note, so the agent can reference how the user spoke. Photos arrive as image attachments → base64 → multimodal message parts — Muse Spark reads them natively.
- **MCP bridge.** Any stdio MCP server listed in the config attaches at runtime; its tools (with full JSON schemas) merge into the model's toolset. Ghostfox brings 72 browser tools. The bridge is the same protocol Muse Spark 1.3 ships native support for.

---

## Quick start

```bash
npm install -g kancilai

# or build from source:
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

- **No audio replies yet** — the Meta Model API catalog has speech-to-text (Muse Voice Transcribe, working in Kancil) but no text-to-speech model, so Kancil replies in text (plus images). All-Meta, no third-party TTS bolted on. Roadmap: voice replies when Meta ships TTS.
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
