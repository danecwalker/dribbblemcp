---
name: dribbble-inspiration
description: >
  Pull UI design inspiration from Dribbble via the dribbble MCP server — search
  shots, inspect images, extract layout/color/typography principles, and apply
  them to original designs. Use when the user asks for design inspiration,
  Dribbble references, UI moodboards, "how do other products design X",
  screenshot research, visual references for dashboards/landing pages/mobile
  flows, or runs /dribbble-inspiration. Complements Mobbin (real product screens)
  with polished concept/shot inspiration from Dribbble.
---

# Dribbble Inspiration

Use the **dribbble** MCP server to find real UI shots, **look at the images**, and turn observations into original design direction. Never treat titles alone as evidence of what a screen looks like.

## When to use

- User wants visual inspiration for a screen, flow, or product surface
- Designing dashboards, SaaS marketing pages, mobile apps, settings, onboarding, pricing, empty states, etc.
- Need color palettes, density references, component composition ideas
- User mentions Dribbble, moodboards, or "show me how others do X"

Prefer **Mobbin** when you need *production* app screenshots and multi-step flows. Prefer **Dribbble** for polished concept work, visual style, and exploratory UI.

## Prerequisites

The `dribbble` MCP server must be configured. Tools:

| Tool | Use for |
|------|---------|
| `search_shots` | Natural-language UI search (default starting point) |
| `search_by_tag` | Known single tags (`dashboard`, `saas`, `mobile-app`) |
| `get_shot` | High-res follow-up on a promising result |

If tools are missing, tell the user to install/configure `dribbblemcp` (see project README).

## Workflow

### 1. Clarify the design target

Capture (ask briefly if missing):

- **Surface**: e.g. analytics dashboard, mobile checkout, marketing hero
- **Domain**: fintech, health, B2B SaaS, consumer social…
- **Constraints**: light/dark, dense/sparse, web/mobile, brand personality

### 2. Search with specific queries

Call `search_shots` with **concrete** language. Run 1–3 focused searches rather than one vague one.

**Good queries**

- `fintech dashboard dark mode charts`
- `SaaS pricing page comparison table`
- `mobile banking home screen cards`
- `AI chat empty state illustration`
- `settings page two-column form`

**Weak queries** (avoid)

- `modern ui`
- `clean design`
- `dashboard` alone when you care about a domain

Parameters:

- `limit`: 4–8 (default 6). Higher burns context.
- `include_images`: leave `true` so you can actually see the UI.

Use `search_by_tag` only for established tags (`dashboard`, `landing-page`, `mobile-app`, `saas`, `ui`, `ux`).

### 3. Look at the images

For each returned shot:

1. Open the visual (MCP image content) — do **not** invent layout from the title.
2. Note structure: grid, hierarchy, primary CTA placement, navigation pattern.
3. Note craft: spacing density, corner radius language, color roles, type scale, elevation/shadows, icon style.
4. Discard mismatches (e.g. illustration-only shots when you need product UI).

When presenting options to the user, **always** markdown-link the shot URL:

```markdown
1. [Vaulta SaaS Dashboard](https://dribbble.com/shots/…) — dense KPI row + left nav; cool neutrals with single accent.
```

### 4. Deep-dive the best refs

Call `get_shot` on 1–3 winners for higher-res images and description/designer metadata. Re-examine details: table density, form layout, empty states, chart treatments.

### 5. Extract principles — then design originally

Produce a short **inspiration brief** the user (or you) can act on:

- **Layout patterns** (e.g. “12-col with sticky left nav; main = KPI strip + 2-col chart/table”)
- **Visual system** (bg/surface/border/accent roles — approximate hex if visible)
- **Typography** (scale steps, weight usage, tabular nums for data)
- **Components** to mirror *structurally* (cards, filters, segment controls)
- **What to avoid** (over-decoration, unusable density, pure concept flair)

Then implement **original** UI. Do not clone shot composition, illustration, or branding. Dribbble work is owned by its designers — use it as a teacher, not a template dump.

### 6. Optional: translate into code / design system

If the user wants implementation:

- Map principles to their stack (Tailwind tokens, CSS variables, component library)
- Prefer system-consistent spacing (4/8pt) over one-off magic numbers
- Keep accessibility (contrast, focus, hit targets) even when refs are glossy

## Output format (default)

When the user asked for inspiration (not full implementation yet):

```markdown
## Inspiration brief: <target>

### References
1. [Title](url) — 1-line visual takeaway
2. …

### Patterns to take
- …
### Patterns to skip
- …
### Proposed direction
- Layout: …
- Color: …
- Type: …
- Signature components: …
```

If they asked you to **build** the UI, fold the brief into a short preamble, then ship the design/code.

## Guardrails

- Always **view images** before describing UI.
- Always **link** Dribbble URLs; credit designers when known.
- Keep request volume modest (no pagination spam).
- If search returns nothing or errors (WAF), retry once with a simpler query; then report failure clearly.
- Never claim a shot is “from product X” unless the shot itself says so — much of Dribbble is conceptual.

## Quick trigger examples

- “Find Dribbble inspo for a dark analytics dashboard”
- “Moodboard for a consumer fintech mobile home”
- “How are people designing AI agent settings pages?”
- `/dribbble-inspiration pricing page for B2B SaaS`
