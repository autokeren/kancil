# Kancil 🦌

**The clever agent that lives in your Messenger.**

All-Meta stack: **Muse Spark 1.3** brain · **Messenger** + **WhatsApp Business** channels · **Muse Image** + **Muse Voice** media. Kancil is an agent harness — you chat, it builds: websites, images, files — and replies with live links to what it made.

Built on the open-source [Gulutux agent engine](https://github.com/autokeren/gulutux) concepts — for this hackathon, the Meta integration (provider, channels, tools, publisher) was built from scratch.

## Quick start

```bash
npm install -g kancil   # (binary wrapper — coming)
kancil                  # 3-step wizard: Meta API key → channels → workspace
kancil chat             # try it locally
kancil serve            # webhooks + live sites
```

## What the demo shows

1. `kancil` wizard → live in under a minute
2. Voice note on Messenger: *"bikinin website kedai kopi gua"*
3. Kancil (Muse Spark 1.3) plans → writes code → publishes a live site
4. Reply arrives in chat with the public link

## Stack

| Layer | Tech |
|---|---|
| Brain | Meta Muse Spark 1.3 (native tool calling) via Meta Model API |
| Channels | Facebook Messenger Platform + WhatsApp Business API (dev mode) |
| Media | Muse Image 1.0 (banners), Muse Voice Transcribe (voice notes) |
| Engine | Go single binary — agent loop, tools, static site server, auto-tunnel |

## Status

Hackathon build — Meta Global AI Developer Hackathon (Oct 27 – Nov 4, 2026).

> Demo runs in Messenger app development mode (standard pre-launch). Production rollout requires Meta's standard app review.
