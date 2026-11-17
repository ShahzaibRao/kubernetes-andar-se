#!/usr/bin/env -S just --justfile
# Root justfile — Phase 1 slides kit (Piotr1215 ke pattern se adapted)
# Har recipe ka naam hi us ka kaam hai. `just --list` se sab dekho.

# Sab recipes dikhao
default:
  @just --list

# Tools check karo — kya kya install hai, kya missing hai
check:
  @echo "--- tools ---"
  @command -v slides >/dev/null && echo "slides: OK ($(slides --help 2>&1 | head -1))" || echo "slides: MISSING (./install.sh chalao)"
  @command -v just >/dev/null && echo "just: OK ($(just --version))" || echo "just: MISSING"
  @command -v tmux >/dev/null && echo "tmux: OK ($(tmux -V))" || echo "tmux: MISSING"
  @command -v figlet >/dev/null && echo "figlet: OK" || echo "figlet: MISSING"
  @command -v graph-easy >/dev/null && echo "graph-easy: OK" || echo "graph-easy: MISSING (diagrams ke liye)"
  @command -v java >/dev/null && echo "java: OK" || echo "java: MISSING (PlantUML ke liye)"
  @test -f /usr/local/bin/plantuml.jar && echo "plantuml.jar: OK" || echo "plantuml.jar: MISSING (PlantUML ke liye)"

# Nayi lesson ka deck banao (template se)
start lesson:
  #!/usr/bin/env bash
  set -euo pipefail
  mkdir -p "{{lesson}}/diagrams"
  cp templates/slides_template.md "{{lesson}}/slides.md"
  chmod +x "{{lesson}}/slides.md"
  echo "Deck ban gaya: {{lesson}}/slides.md"
  echo "Ab DECK-INDEX.md mein us lesson ki script kholo aur slides bharo."

# Kisi lesson ka deck present karo (slides kholo)
# cd zaroori hai: theme ../theme.json deck folder se relative hai
present lesson:
  @cd {{lesson}} && slides slides.md

# Design A runner build karo (Go + Bubble Tea TUI)
build-presenter:
  cd presenter && go build -o "$HOME/.local/bin/k8s-present" .

# Design A runner se lesson chalao (deck.md format)
present-a lesson:
  @"$HOME/.local/bin/k8s-present" {{lesson}}/deck.md

# PlantUML diagram ko ASCII mein render karo (diagrams/<name>.puml -> .utxt)
plantuml diagram:
  #!/usr/bin/env bash
  if command -v java >/dev/null 2>&1 && [ -f /usr/local/bin/plantuml.jar ]; then
    java -jar /usr/local/bin/plantuml.jar "diagrams/{{diagram}}.puml" -utxt
    cat "diagrams/{{diagram}}.utxt"
  else
    echo "PlantUML nahi mila (java + /usr/local/bin/plantuml.jar chahiye)"
  fi

# graph-easy diagram ko box art mein render karo (diagrams/<name>.dot -> .txt)
digraph diagram:
  #!/usr/bin/env bash
  if command -v graph-easy >/dev/null 2>&1; then
    graph-easy "diagrams/{{diagram}}.dot" --as=boxart > "diagrams/{{diagram}}.txt"
    cat "diagrams/{{diagram}}.txt"
  else
    echo "graph-easy nahi mila (sudo apt install libgraph-easy-perl)"
  fi

# Kisi lesson ka deck reset karo (us ki reset.sh chalao, agar hai)
reset lesson:
  @cd {{lesson}}; [ -x ./reset.sh ] || { echo "no reset.sh in {{lesson}}" >&2; exit 1; }; ./reset.sh

# Bara title figlet se dikhao (title slide ke liye)
intro_toilet *title:
  @echo "{{title}}" | figlet -f small -w 90
