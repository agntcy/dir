---
name: agntcy-dir
description: "Use this skill to interact with an AGNTCY Directory (DIR) instance. This is a bootstrap: it installs the complete agntcy-dir skill (setup, authoring, publishing, discovery, verification, sync, install workflows) from the AGNTCY Directory GitHub repository, then defers to it."
metadata:
  author: AGNTCY Contributors
---

# AGNTCY Directory — skill bootstrap

This file intentionally does **one** thing: install the full `agntcy-dir`
skill from the AGNTCY Directory repository and hand over to it. Do not act on
Directory requests from this bootstrap alone — install the full skill first,
then follow its instructions.

The complete skill lives in the repository's
[`.skill/` folder](https://github.com/agntcy/dir/tree/main/.skill): a router
`SKILL.md` plus seven reference files.

## Install

Copy the folder into the host's skills directory, named exactly `agntcy-dir`
(the folder name must match the skill's frontmatter `name`):

| Host                        | Destination                                                    |
| --------------------------- | -------------------------------------------------------------- |
| GitHub Copilot (user-wide)  | `~/.copilot/skills/agntcy-dir/`                                |
| VS Code Copilot (workspace) | `.github/skills/agntcy-dir/`                                   |
| Claude Code                 | `.claude/skills/agntcy-dir/` or `~/.claude/skills/agntcy-dir/` |
| Other hosts                 | that host's skills directory, same folder name                 |

Fetch from GitHub (raw content, `main` branch):

```bash
BASE=https://raw.githubusercontent.com/agntcy/dir/main/.skill
DEST=~/.copilot/skills/agntcy-dir        # adjust to the host (table above)
mkdir -p "$DEST/references"
curl -fsSL "$BASE/SKILL.md" -o "$DEST/SKILL.md"
for f in authoring discovery install publishing setup sync verification; do
  curl -fsSL "$BASE/references/$f.md" -o "$DEST/references/$f.md"
done
```

PowerShell (Windows):

```powershell
$BASE = 'https://raw.githubusercontent.com/agntcy/dir/main/.skill'
$DEST = "$HOME\.copilot\skills\agntcy-dir"   # adjust to the host (table above)
New-Item -ItemType Directory -Force -Path "$DEST\references" | Out-Null
Invoke-WebRequest "$BASE/SKILL.md" -OutFile "$DEST\SKILL.md"
foreach ($f in 'authoring','discovery','install','publishing','setup','sync','verification') {
  Invoke-WebRequest "$BASE/references/$f.md" -OutFile "$DEST\references\$f.md"
}
```

Rules:

- Ask the user for the install scope (workspace vs user profile) when the
  host supports both.
- If `agntcy-dir` is already installed, confirm before overwriting it.
- Verify the install: the destination must contain `SKILL.md` and all seven
  `references/*.md` files; report the written paths.

## After installing

Reload skills if the host requires it (e.g. start a new chat turn or restart
the session), then handle the user's Directory request by following the
installed skill — not this bootstrap.
