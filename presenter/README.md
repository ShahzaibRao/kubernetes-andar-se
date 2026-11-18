# k8s-present — Design A (Split Console) Runner

Rao ne Design A choose kiya (2026-11-17), aur **Design v2 approve kiya
(2026-11-18)**. Ye `slides` ka replacement hai — slides ka Ctrl+E poora
code block ek saath chalata tha aur progress bar kabhi nahi dikhata tha
(output pipe mein capture hota hai, TTY nahi).

## Layout (v2)

```
Lesson 03 — containerd  ·  Slide 5/19
────────────────────────────────────────────
Step 2 — Config File Banao        ╭─ Terminal ─────────────────╮
HINT › Default config banao ...   │ $ sudo mkdir -p ... ✓ exit 0│
• ...                             │ $ wc -l ... ✗ exit 1        │
Commands — ↑↓ select · Enter run: │ wc: ... No such file        │
▶ containerd config default | ... │ › $ echo hello ✓ exit 0     │
  wc -l /etc/containerd/config... │ hello                       │
                                  │ › Tab ya / dabao — apni ... │
```

- **Left (PINNED/static):** heading, HINT, bullets, command list — ye
  kabhi hilta nahi, chahe right side mein download chal raha ho. Left
  pane sirf slide badalne par dobara render hota hai (cached).
- **Right (Terminal):** har command ka apna **output block** — `$ cmd`
  header + status chip (…running / ✓ exit 0 / ✗ exit N) + output.
  Blocks **poori session mein mehfooz rehte hain**, slide badalne par
  bhi pichla output gayab nahi hota (purane finished blocks 2000 lines
  ke baad katte hain).
- **Free terminal input:** neeche `› ` wali line — **Tab ya `/`**
  dabao, apni command type karo, Enter se chalao. Ye slide ki list
  mein add nahi hoti, alag block ban ke chalti hai.
- Commands **pty** ke through chalti hain, is liye `curl` ki progress
  bar aur `ctr pull` ki progress bilkul waise dikhti hai jaise terminal
  mein chalao

## Keys

| Key | Kaam |
|---|---|
| ↑ / ↓ (ya k / j) | Command select karo |
| Enter | Selected command chalao (ek-ek karke) |
| Tab ya `/` | Free terminal input par jao — apni command likho |
| Enter (input mein) | Apni command chalao |
| Esc (input mein) | Wapas command list par |
| Backspace / Ctrl+U (input mein) | Mitana / saaf karna |
| Space / → | Agla slide |
| ← | Pichla slide |
| Ctrl+C (command chal rahi ho) | Command roko |
| q | Quit |

Command chal rahi ho to keys us command ko forward hoti hain (masalan
`sudo` password type kar sakte ho). Pehle `sudo -v` chala lena behtar hai.

## v1 bugs jo v2 mein fix hain (2026-11-18)

1. **Left pane gayab ho jata tha:** right box 1 column zyada chaura
   tha (border+padding count nahi hui thi) — poora render wrap ho ke
   header scroll ho jata tha. Ab box ki height/width border ke hisab
   se sahi hai (header nazar aata hai).
2. **ANSI escapes leak:** child process ki ANSI sequences screen ko
   kharab karti thin — output ab store karne se pehle sanitize hota
   hai.
3. **`\r` flood:** progress ki har redraw nayi line banati thi —
   `\r` ab line ko overwrite karta hai, jaise asal terminal mein.
4. **Output gayab:** sirf aakhri command ka output rehta tha — ab har
   command ka block session bhar rehta hai.
5. **Flicker:** output chunks ab 80ms par batch ho ke flush hote
   hain, har chunk par full re-render nahi hota.
6. **Lambi header wrap:** lipgloss Width border/padding ko andar
   ginta hai, is liye text `outWidth-4` par truncate hota hai aur
   pty bhi usi chauraai par set hota hai.

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
