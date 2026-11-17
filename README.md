# Terminal Slides — Phase 1 Presentation Kit
## Kubernetes Andar Se — Chapter 0 / Bootstrap

## Stack (kya use ho raha hai, aur kyun)

| Tool | Kaam | Kyun yehi |
|---|---|---|
| **slides** (maaslalani/slides) | Markdown se terminal presentation, `Ctrl+E` se code block execute, output slide par dikhta hai | Video wala stack. Execution **explicit** hai — tum dabao ge tabhi command chalegi. Yehi chahiye, kyunki lab commands ke side effects hain (cert dobara banao ge to purana toot jaye ga). Install verify ho gaya: `go install github.com/maaslalani/slides@latest` |
| **just** (casey/just) | Command runner — har lesson ka `justfile`, har step ek recipe | Har command pehle se likhi, naam se chalti hai: `just check`, `just reset`. Koi typing nahi, koi copy-paste nahi. Install verify ho gaya: v1.59.0 prebuilt binary |
| **tmux** | Live demo ke liye alag session | Slides ek session mein, demo doosre mein. `tmux switchc -t demo` se switch karo. Recording ke doran slides band nahi karni parti |
| **graph-easy** | `.dot` diagram → ASCII box art | Slides mein diagram ASCII mein dikhta hai, image ki zaroorat nahi. `just digraph <name>` |
| **PlantUML** | `.puml` diagram → ASCII (`-utxt`) | Sequence/flow diagrams design karo, ASCII export karo, slide mein dikhao. `just plantuml <name>` |
| **figlet** | Title slide ke bade letters | `just intro_toilet "Title"` |

## slides vs presenterm — imandari se farq

Piotr ka **purana** stack (video wala) `slides` hai. Us ka **naya** stack `presenterm` hai — is mein code block khud chal jata hai jab slide aati hai (`+exec` / `+exec_replace`), aur images bhi dikhti hain.

Rao ke course ke liye **slides behtar hai**, is liye:
- `slides` mein tum `Ctrl+E` dabao ge **tabhi** command chalegi. `presenterm` mein slide aate hi command chal jati hai — aur wapas jaane par dobara chal sakti hai. Lab mein `openssl genrsa` dobara chala to purane certs toot jayen ge. Explicit control yahan safety hai, kami nahi.
- `slides` install verify ho gaya hai (is VM par `go install` se chala, `--help` dekha).
- `presenterm` upgrade path hai — jab decks pakke ho jayen, tab `+exec` wali copy banana aasan hai. Pehle `slides` se recording shuru karo.

## Recording flow — ek lesson kaise record hoti hai

1. VM par jao (Proxmox lab VM — wahi jahan cluster bann raha hai).
2. `just present lesson-03-containerd` — deck khulta hai.
3. Har slide par: pehle **samjhao** (Roman Urdu), phir `Ctrl+E` dabao — command chalti hai, output slide par aata hai.
4. Fail wali slide par pehle fail dikhao (command chalao, error aaye), phir agle slide par fix.
5. Lambi demo ke liye: `tmux switchc -t demo` se live terminal par jao, wahan `just <recipe>` chalao, phir wapas slides par.
6. Recording khatam: `just reset lesson-03-containerd` — lesson ki cheezein saaf karo taake dobara record kar sako.

## Keys (slides ke andar)

- Agli slide: `Space` / `Right` / `Enter` / `n` / `j`
- Pichli slide: `Left` / `Up` / `p` / `h` / `k`
- Code execute karo: `Ctrl+E` (output slide ke end par dikhe ga)
- Kisi slide par jao: `<number>G` (masalan `5G` = slide 5)
- Search: `/` phir term, `Enter` — `Ctrl+N` se agla result
- Pehli slide: `g` `g` — Aakhri slide: `G`
- Bahar niklo: `q` ya `Ctrl+C`

## Folder layout

```
slides/
  README.md                  <- ye file (design + flow)
  install.sh                 <- sab tools install karta hai (VM par ek dafa)
  justfile                   <- root recipes: check, start, present, plantuml, digraph, reset
  theme.json                 <- slides ka theme (Piotr ke theme se adapted)
  templates/
    slides_template.md       <- nayi lesson ka deck is se banta hai
  diagrams/                  <- shared diagrams (.dot / .puml source + rendered .txt)
  lesson-01-welcome/
    slides.md                <- Lesson 01 ka deck (poora)
  lesson-03-containerd/
    slides.md                <- Lesson 03 ka deck (poora, test karne ke liye)
    justfile                 <- Lesson 03 ki recipes (install, check, reset)
  phase1-scripts/            <- Phase 1 ke 3 scripts + roadmap (decks ka source)
  DECK-INDEX.md              <- 13 lessons ka map: kaunsa deck, kahan se bharna hai
```

## Ek nayi lesson ka deck kaise banao

```bash
just start lesson-04-kubelet
```

Ye `templates/slides_template.md` se `lesson-04-kubelet/slides.md` banata hai, phir `DECK-INDEX.md` mein us lesson ki script ka section kholo aur slides bharo. Har lesson ka pattern ek hi hai: **Title → Kyun → Diagram → Fail → Command → Check → Summary**.

## Zaroori notes (recording se pehle)

- `sudo -v` recording se pehle chala lo — sudo password cache ho jaye ga, beech mein password prompt recording kharab nahi kare ga.
- Deck ki file executable honi chahiye (pre-processing ke liye): `chmod +x slides.md`.
- `slides` file change dekh kar live reload karta hai — deck edit karo to presentation khud update ho jati hai.
- Har code block **ek step** hai. Ek block mein 2 steps mat likho — `Ctrl+E` poora block chalata hai.
- Commands mein `<...>` wali jagah (masalan `<worker-2-IP>`) pehle se apne lab ke asli value se bhar lo — recording ke doran placeholder mat chhoro.
