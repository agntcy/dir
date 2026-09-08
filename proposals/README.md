# Proposals

Design documents for changes too large to explain in a pull request
description. A proposal captures the problem, the constraints discovered while
investigating the code, the options considered, and the open decisions — so that
the reasoning survives past the discussion that produced it.

These are **not** user documentation. Published docs live in `docs/content` and
are built with mkdocs; nothing here is part of the docs site.

A proposal is a folder, `proposals/<topic>/`, containing at least a `README.md`
that states its status in the first lines. Status is one of:

- **draft** — under discussion, nothing implemented, subject to change
- **accepted** — agreed direction; implementation may be in progress or partial
- **implemented** — shipped; kept for the reasoning, may drift from the code
- **rejected** / **superseded** — kept deliberately, with a note explaining why
  and a pointer to what replaced it

Proposals are merged rather than left on a branch: the decisions and the
discarded alternatives are the valuable part, and they need somewhere durable to
live. A proposal being merged means "this is a real record of a discussion", not
"this is approved" — that is what the status line is for.

| Proposal | Status |
|---|---|
| [`quality-gate`](quality-gate/) — policy engine for admission and retention of records | draft |
