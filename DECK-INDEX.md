# Phase 1 — Deck Index (13 lessons → 13 decks)
## Kubernetes Andar Se — Chapter 0 / Bootstrap

**Format (2026-11-17, Rao's call — Design A / Split Console):** decks ab
`k8s-present` runner ke `deck.md` format mein hain (`presenter/README.md`
dekho). Purani `slides.md` files abhi repo mein hain, lekin nayi recording
Design A se hogi. Lesson 03 ka `deck.md` pehla converted deck hai.

**Design v2 (2026-11-18, Rao approved):** runner v2 — left pane pinned/static,
har command ka output block session bhar mehfooz, neeche free terminal input
(Tab ya `/` se apni command), progress bar sirf right pane mein, ANSI + `\r`
sanitize, 80ms batched flush. v1 ke bugs (left pane gayab, output gayab,
header scroll) fix hain — `presenter/README.md` mein "v1 bugs" section dekho.

**Rule:** 1 lesson = 1 deck = 1 video (agar Rao module-wise chahe to 4 decks mein merge karna aasan hai — deck files ko `cat` karke jor do, `---` separator pehle se hai).

**Status:** Lesson 01 aur 03 ke decks poore hain (test karne ke liye). Baaki decks `just start <lesson>` se banao, phir neeche wali script se bharo.

| # | Lesson | Deck folder | Source script | Status |
|---|---|---|---|---|
| 01 | Welcome | `lesson-01-welcome/` | Script 01, Lesson 01 | ✅ Poora |
| 02 | Tips & Tricks | `lesson-02-tips-tricks/` | Script 01, Lesson 02 | ⬜ Template se banao |
| 03 | containerd | `lesson-03-containerd/` | Script 01, Lesson 03 | ✅ Poora |
| 04 | kubelet | `lesson-04-kubelet/` | Script 01, Lesson 04 | ⬜ Template se banao |
| 05 | etcd | `lesson-05-etcd/` | Script 02, Lesson 05 | ⬜ Template se banao |
| 06 | kube-apiserver | `lesson-06-apiserver/` | Script 02, Lesson 06 | ⬜ Template se banao |
| 07 | kube-scheduler | `lesson-07-scheduler/` | Script 02, Lesson 07 | ⬜ Template se banao |
| 08 | kube-controller-manager | `lesson-08-controller-manager/` | Script 02, Lesson 08 | ⬜ Template se banao |
| 09 | Joining Nodes | `lesson-09-joining-nodes/` | Script 02, Lesson 09 | ⬜ Template se banao |
| 10 | Network (Flannel) | `lesson-10-network/` | Script 03, Lesson 10 | ⬜ Template se banao |
| 11 | kube-proxy | `lesson-11-kube-proxy/` | Script 03, Lesson 11 | ⬜ Template se banao |
| 12 | CoreDNS | `lesson-12-coredns/` | Script 03, Lesson 12 | ⬜ Template se banao |
| 13 | Conclusion | `lesson-13-conclusion/` | Script 03, Lesson 13 | ⬜ Template se banao |

## Har deck bharne ka pattern (template mein pehle se hai)

1. **Title** — ek line: is lesson ke baad viewer kya kar sakega
2. **Kyun?** — viewer ka masla, jo ye lesson hal karta hai
3. **Diagram** — ASCII diagram (PlantUML `-utxt` ya graph-easy `--as=boxart` ya haath se)
4. **FAIL** — pehle woh command jo fail hogi, error dikhao
5. **Command** — har step ek alag code block (ek `Ctrl+E` = ek step)
6. **Check** — output mein kya dekhna hai
7. **Summary** — kya seekha + dependency line (kis par dependent hai, kiske baghair kya toot-ta hai)

## Diagrams — kahan se aayen ge

| Diagram | Source | Render |
|---|---|---|
| Request flow (kubectl → apiserver → etcd → scheduler → kubelet → containerd → runc) | `diagrams/request-flow.puml` | `just plantuml request-flow` |
| Component dependencies (kaun kis par dependent hai) | `diagrams/dependencies.dot` | `just digraph dependencies` |
| Pod network (Flannel VXLAN, cni0, flannel.1) | `diagrams/pod-network.puml` | `just plantuml pod-network` |
| DNS flow (Pod → CoreDNS → ClusterIP → kube-proxy → Pod) | `diagrams/dns-flow.puml` | `just plantuml dns-flow` |
| TLS bootstrapping (token → CSR → approve → sign → cert) | `diagrams/bootstrap.puml` | `just plantuml bootstrap` |

Render ki hui `.txt` / `.utxt` file ko slide mein code block mein paste karo — ya slides ki pre-processing (`~~~just plantuml <name>~~~`) se live render karwao. Pre-processing ke liye `slides.md` executable honi chahiye (`chmod +x slides.md`).
