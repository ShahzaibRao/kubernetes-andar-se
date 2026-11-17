#!/usr/bin/env bash
# Terminal Slides kit — install script (lab VM par ek dafa chalao)
# Installs: slides (Go), just 1.59.0 (prebuilt), tmux, figlet,
#           graph-easy (Perl), plantuml.jar (Java chahiye)
# Verify hua: slides `go install` se + just 1.59.0 binary is VM par chal gaye.
set -euo pipefail

JUST_VERSION="1.59.0"
BIN_DIR="${HOME}/.local/bin"
mkdir -p "${BIN_DIR}"

say() { printf '\n==> %s\n' "$*"; }

# --- just ---
if command -v just >/dev/null 2>&1; then
  say "just pehle se hai: $(just --version)"
else
  say "just ${JUST_VERSION} install ho raha hai..."
  tmp="$(mktemp -d)"
  curl -fsSLO --output-dir "${tmp}" \
    "https://github.com/casey/just/releases/download/${JUST_VERSION}/just-${JUST_VERSION}-x86_64-unknown-linux-musl.tar.gz"
  tar xzf "${tmp}/just-${JUST_VERSION}-x86_64-unknown-linux-musl.tar.gz" -C "${tmp}"
  install -m 755 "${tmp}/just" "${BIN_DIR}/just"
  rm -rf "${tmp}"
  say "just install ho gaya: $("${BIN_DIR}/just" --version)"
fi

# --- slides (Go chahiye) ---
if command -v slides >/dev/null 2>&1; then
  say "slides pehle se hai: $(slides --help 2>&1 | head -1)"
else
  if ! command -v go >/dev/null 2>&1; then
    echo "Go nahi mila. Pehle Go 1.23+ install karo (https://go.dev/dl), phir dobara chalao."
    echo "Ubuntu par: sudo apt install golang-go  (version check karo: go version)"
    exit 1
  fi
  say "slides install ho raha hai (go install)..."
  go install github.com/maaslalani/slides@latest
  say "slides install ho gaya: $(command -v slides || echo "${HOME}/go/bin/slides")"
fi

# --- tmux + figlet (apt) ---
if command -v tmux >/dev/null 2>&1 && command -v figlet >/dev/null 2>&1; then
  say "tmux + figlet pehle se hain"
else
  say "tmux + figlet install ho rahe hain (apt)..."
  sudo apt update && sudo apt install -y tmux figlet
fi

# --- graph-easy (Perl, diagrams ke liye) ---
if command -v graph-easy >/dev/null 2>&1; then
  say "graph-easy pehle se hai"
else
  say "graph-easy install ho raha hai (apt libgraph-easy-perl, warna cpan)..."
  if sudo apt install -y libgraph-easy-perl 2>/dev/null; then
    say "graph-easy install ho gaya"
  else
    echo "apt se nahi hua — cpan se try karo: cpan Graph::Easy"
  fi
fi

# --- plantuml (Java chahiye) ---
if [ -f /usr/local/bin/plantuml.jar ]; then
  say "plantuml.jar pehle se hai"
else
  if ! command -v java >/dev/null 2>&1; then
    echo "Java nahi mila — PlantUML ke liye Java 11+ chahiye: sudo apt install -y default-jre"
    echo "Phir: curl -fsSLO https://github.com/plantuml/plantuml/releases/download/v1.2024.7/plantuml-1.2024.7.jar -o /usr/local/bin/plantuml.jar"
  else
    say "plantuml.jar download ho raha hai..."
    sudo curl -fsSL https://github.com/plantuml/plantuml/releases/download/v1.2024.7/plantuml-1.2024.7.jar \
      -o /usr/local/bin/plantuml.jar
    say "plantuml.jar install ho gaya"
  fi
fi

say "Sab ho gaya. Check karo:"
say "  just --version && slides --help | head -3 && tmux -V"
say "PATH mein ${BIN_DIR} aur \$(go env GOPATH)/bin hone chahiye."

# --- PATH fix (2026-11-17: Rao ki machine par ~/go/bin PATH mein nahi tha) ---
# slides ~/go/bin mein install hoti hai, just ~/.local/bin mein — dono ko
# ~/.bashrc mein daal do taake har nayi terminal mein milen.
for line in 'export PATH="$HOME/go/bin:$PATH"' 'export PATH="$HOME/.local/bin:$PATH"'; do
  grep -qxF "$line" "${HOME}/.bashrc" 2>/dev/null || echo "$line" >> "${HOME}/.bashrc"
done
export PATH="${HOME}/go/bin:${BIN_DIR}:${PATH}"
say "PATH fix ho gaya (~/.bashrc mein add kar diya). Nayi terminal kholo ya chalao: source ~/.bashrc"
