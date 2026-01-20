# About this project

This is a vibe-coding project. 

Almost everything in this repo is not my work, so be cautious about it.

## Motivation

I want to spend my days while my phone's powered off but I don't want to miss important messages from my friends.

I wanted a TUI iMessage client on my Linux computer, just for myself.

## You want LLM generated README?

The following is the LLM generated README content:

----

# iMessage Lightweight CLI (WIP)

This workspace hosts a personal iMessage CLI client for Linux plus a macOS registration generator.

## Layout
- `mac-registration-provider/`: Generate `registration-data.json` on macOS. Single-shot `--out` flow.
- `imessage-client/`: Go CLI for Linux. Commands:
  - `check-messages` (poll unread; uses `--registration` and `--store`).
  - `send` (stubbed send flow; same flags).
- `docs/`: Planning and usage notes.

## Quickstart
1) On macOS:
```bash
./mac-registration-provider --out registration-data.json
```
Copy the JSON to your Linux box.

2) On Linux (poll):
```bash
./imessage-client check-messages \
  --registration /path/to/registration-data.json \
  --store ${XDG_CONFIG_HOME:-$HOME/.config}/imessage-client/state.json
```

3) Send (stubbed):
```bash
./imessage-client send --chat SOME_ID "hello"
```

## Status
- Registration generator trimmed to single output flow.
- Client scaffolding in place (commands, config, storage). Handshake and transport are not yet implemented; commands will print friendly messages when stubs are hit.

See [docs/migration-plan.md](docs/migration-plan.md) and [docs/handshake-plan.md](docs/handshake-plan.md) for porting details.
