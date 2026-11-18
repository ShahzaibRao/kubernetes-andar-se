# k8s-present — Design A (Split Console) Runner

Rao ne Design A choose kiya (2026-11-17), **Design v2 approve kiya
(2026-11-18)**, aur **Design v3 (loaded slides) approve kiya (2026-11-18)**.
Ye `slides` ka replacement hai — slides ka Ctrl+E poora code block ek
saath chalata tha aur progress bar kabhi nahi dikhata tha (output pipe
mein capture hota hai, TTY nahi).

## Layout (v2 + v3)

```
Kubernetes Andar Se  ·  Lesson 03 — containerd  ·  Slide 9/21
────────────────────────────────────────────
Pehla container — FAIL hona chahiye ╭─ Terminal ────────────────╮
HINT › `ctr` image khud pull ...   │ $ sudo ctr images pull ... │
• `ctr` containerd ka apna CLI ... │ $ sudo ctr run ... ✗ exit 1│
EXPECT › exec: "runc": ...         │ ctr: failed to create ...  │
Commands — ↑↓ select · Enter run:  │ › Tab ya / dabao — apni ...│
  # pehle image pull karo ...      │                            │
▶ sudo ctr images pull ...         │                            │
```

- **Left (PINNED/static):** heading, HINT, bullets, EXPECT, command
  list — ye kabhi hilta nahi, chahe right side mein download chal
  raha ho. Left pane sirf slide badalne par dobara render hota hai
  (cached).
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

## v3 — loaded slides (2026-11-18)

Rao ka feedback: slides chal to gayin, lekin professional nahi lag
rahin — CNCF talks (Piotr Zaniewski, presenterm) jaisa feel nahi.
Research se jo nikla:

- **Title = takeaway, label nahi.** "Step 1 — Install Karo" ki jagah
  "Install: pehle iptables, phir binaries" — title parhte hi pata
  chale ke is slide ka nateeja kya hai.
- **Har slide par say-line (`> `).** Ek jumla jo bolna hai — WHY.
  Phase 1 scripts ki "Bolne wali line" yahin aati hai.
- **EXPECT (`! `).** Kamyabi kaisi dikhti hai — `Active: active
  (running)`, `SystemdCgroup = true`, ya FAIL wali slide par exact
  error. Command chalane se pehle pata ho ke kya dekhna hai.
- **`# ` annotation.** ` ```sh ` block ke andar `# ` wali line
  command nahi — woh us ke upar dim dikhti hai, code khud samjhata
  hai (jaise Piotr ke blocks mein comments hote hain).
- **`**bold**`.** Bullet/paragraph mein `**...**` bold hota hai —
  keywords (KYA/KAISE, manager/executor) nazar aate hain.
- **Title slide + closing slide.** Pehli slide agenda, aakhri slide
  agli lesson + parhne ke links.
- **Rhythm: concept → command → proof.** Concept slides (0 commands)
  aur verify slides (curl/ps) command slides ke beech mein.
- Header mein series branding: `Kubernetes Andar Se · Lesson … ·
  Slide N/M`.

## Deck format (deck.md)

```markdown
# Lesson 03 — containerd

## Install: pehle iptables, phir binaries
> CNI bridge ko NAT ke liye iptables chahiye — ye step miss hua to aakhir mein `nerdctl run` fail hoga.
- **iptables** host tool hai — containerd ka hissa nahi, lekin CNI ke baghair kaam nahi
! bin/ folder nikla — containerd, ctr, containerd-shim-runc-v2
```sh
# host tool — CNI bridge ke liye zaroori
sudo apt install -y iptables
# tarball se binaries nikalo
tar xzvof containerd-2.2.1-linux-amd64.tar.gz
```
```text
diagram yahan — ye execute NAHI hota
```
```

- `# ` lesson title, `## ` naya slide (takeaway sentence)
- `> ` HINT — bolne wali line, WHY ek jumle mein (har slide par)
- `! ` EXPECT — kamyabi kaisi dikhe gi (ya FAIL slide par exact error)
- `- ` bullet, plain text paragraph — dono mein `**bold**` chalta hai
- ` ```sh ` block ki **har line ek alag command** hai (↑↓ se select,
  Enter se run); `# ` line us ke upar dim annotation hai, run nahi hoti
- ` ```text ` diagram hai — dikhta hai, chalta nahi (lambi line
  wrap ho jati hai — diagram **vertical** rakho, ~60 columns se
  chhota)
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
