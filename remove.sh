#!/usr/bin/env bash
# Terminal Slides kit — remove script (install.sh ka ulta)
# Removes what install.sh installed:
#   - slides binary (~/go/bin/slides)
#   - just binary (~/.local/bin/just)
#   - plantuml.jar (/usr/local/bin/plantuml.jar)
#   - PATH lines from ~/.bashrc
# apt packages (tmux, figlet, libgraph-easy-perl) sirf --with-apt flag se
# remove hote hain, kyunki woh tumhare kaam ke bhi ho sakte hain.
# Usage:
#   ./remove.sh            # sirf binaries + jar + PATH lines
#   ./remove.sh --with-apt # + apt packages bhi remove karo
set -euo pipefail

say() { printf '\n==> %s\n' "$*"; }

# --- binaries + jar ---
for f in "${HOME}/go/bin/slides" "${HOME}/.local/bin/just" /usr/local/bin/plantuml.jar; do
  if [ -e "$f" ]; then
    rm -f "$f"
    say "removed: $f"
  else
    say "nahi mila (skip): $f"
  fi
done

# --- PATH lines ~/.bashrc se nikalo (backup ke saath) ---
BASHRC="${HOME}/.bashrc"
if [ -f "${BASHRC}" ]; then
  if grep -qF 'export PATH="$HOME/go/bin:$PATH"' "${BASHRC}" \
    || grep -qF 'export PATH="$HOME/.local/bin:$PATH"' "${BASHRC}"; then
    cp "${BASHRC}" "${BASHRC}.bak"
    grep -vF 'export PATH="$HOME/go/bin:$PATH"' "${BASHRC}.bak" \
      | grep -vF 'export PATH="$HOME/.local/bin:$PATH"' > "${BASHRC}"
    say "PATH lines ~/.bashrc se nikal di (backup: ~/.bashrc.bak)"
  else
    say "~/.bashrc mein PATH lines nahi mili (skip)"
  fi
fi

# --- apt packages (sirf flag se) ---
if [ "${1:-}" = "--with-apt" ]; then
  say "apt packages remove ho rahe hain..."
  sudo apt remove -y tmux figlet libgraph-easy-perl
  sudo apt autoremove -y
fi

say "Ho gaya. Is terminal mein PATH abhi bhi purana ho sakta hai — nayi terminal kholo ya chalao: source ~/.bashrc"
say "Repo folder (kubernetes-andar-se) ye script nahi hatati — us ko khud delete karo agar chahiye."
