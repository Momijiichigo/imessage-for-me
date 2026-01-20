# Usage

## Generate registration on Mac
```bash
./mac-registration-provider --out registration-data.json
```
Copy `registration-data.json` to your Linux machine (keep it private).

## Poll for unread messages (Linux)
```bash
./imessage-client check-messages \
  --registration /path/to/registration-data.json \
  --store ${XDG_CONFIG_HOME:-$HOME/.config}/imessage-client/state.json
```
- `--store ""` keeps state in-memory (no persistence between runs).
- If registration is expired or missing validation data, the command exits with an error so cron/systemd can alert you.

## Send (stub)
```bash
./imessage-client send --chat SOME_ID "hello world"
```
Send is currently a stub; it will validate registration and then return a friendly message until transport is wired.

## Interactive (placeholder)
```bash
./imessage-client
```
Interactive mode is stubbed; once transport is wired, it will establish a live session for send/receive.
