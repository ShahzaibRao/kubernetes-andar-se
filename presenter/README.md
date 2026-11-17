# k8s-present — Design A (Split Console) Runner

Rao ne Design A choose kiya (2026-11-17). Ye `slides` ka replacement hai —
slides ka Ctrl+E poora code block ek saath chalata tha aur progress bar
kabhi nahi dikhata tha (output pipe mein capture hota hai, TTY nahi).

## Layout

```
Lesson 03 — containerd  ·  Slide 5/19
────────────────────────────────────────────
Step 2 — Config File Banao        │ Output — $ containerd config ...
› Default config banao ...        │ ─────────────────────────────
• ...                             │ 128
Commands — ↑↓ select · Enter run: │
▶ containerd config default | ... │ (output yahan stream hota hai)
  wc -l /etc/containerd/config... │
```

- **Left:** heading, quote, bullets, command list
- **Right:** output pane — jo command chalao, us ka output us ke naam ke
  saath, live stream hota hai
- Commands **pty** ke through chalti hain, is liye `curl` ki progress
  bar aur `ctr pull` ki progress bilkul waise dikhti hai jaise terminal
  mein chalao

## Keys

| Key | Kaam |
|---|---|
| ↑ / ↓ (ya k / j) | Command select karo |
| Enter | Selected command chalao (ek-ek karke) |
| Space / → | Agla slide |
| ← | Pichla slide |
| Ctrl+C (command chal rahi ho) | Command roko |
| q | Quit |

Command chal rahi ho to keys us command ko forward hoti hain (masalan
`sudo` password type kar sakte ho). Pehle `sudo -v` chala lena behtar hai.

## Deck format (deck.md)

```markdown
# Lesson 03 — containerd

## Step 1 — containerd Install Karo
> Tarball se binary nikalo aur /usr/local/bin mein daalo
- bullet point
```sh
curl -fsSLO https://example.com/file.tar.gz
tar xzvof file.tar.gz
```
```text
diagram yahan — ye execute NAHI hota
```
```

- `# ` lesson title, `## ` naya slide
- `> ` quote, `- ` bullet, plain text paragraph
- ` ```sh ` block ki **har line ek alag command** hai (↑↓ se select,
  Enter se run)
- ` ```text ` diagram hai — dikhta hai, chalta nahi
- **Note:** har command alag `bash -c` mein chalti hai, is liye jo
  cheez agle command ko chahiye (masalan `VAR=$(...)`) us ko **ek hi
  line** mein likho: `VAR=$(...); curl "http://${VAR}:9898"`

## Build aur run

```bash
# repo root se:
just build-presenter          # ~/.local/bin/k8s-present banata hai
just present-a lesson-03-containerd

# ya seedha:
cd presenter && go build -o ~/.local/bin/k8s-present .
k8s-present ../lesson-03-containerd/deck.md

# deck check (UI khole baghair):
k8s-present --check ../lesson-03-containerd/deck.md
```

Commands deck file ki directory mein chalti hain (downloads wahin
aate hain).
