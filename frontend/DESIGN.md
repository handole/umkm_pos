# Design Direction: UMKM POS Web Backoffice (Draft Without External Brand Guide)

> Status: **Draft without user-supplied direction** (R-37). Built with explicit dials and recorded decisions (R-31).

## Three Dials (Part 3)
- **ENERGY: 2 (Balanced)** — Operational retail software that feels grounded and intentional, not loud or sterile.
- **RHYTHM: 2 (Balanced)** — Asymmetric two-column split on desktop (context ledger panel + focused auth form), collapsing cleanly to a single-column form-first flow on tablet and mobile.
- **MOTION: 1 (Calm)** — Crisp state transitions (`150ms`) on hover, focus, and tab switches only; disabled under `prefers-reduced-motion: reduce`.

## Palette (R-29: 2 Core + 1 Accent + Neutrals)
- **Neutral Paper (`#FAF8F5`)**: Warm alabaster background to reduce glare during long store shifts.
- **Surface (`#FFFFFF`) & Border (`#D6D0C4`)**: Solid paper cards with crisp 1px ledger borders instead of floating soft shadows.
- **Core Ink (`#1C1917`) & Secondary (`#57534E`)**: High-contrast charcoal text (16.5:1 and 7.2:1 on `#FAF8F5`).
- **Core Brand (`#1E3A2F`)**: Deep merchant forest green for primary headings, active tabs, and primary actions (12.34:1 with white text).
- **Accent (`#C2410C`)**: Terracotta accent reserved strictly for key operational highlights and active focus/CTA states (5.18:1 with white text).
- **Semantic Error (`#B91C1C` on `#FEF2F2`)**: High-contrast error surface paired with explicit error text and icon (5.91:1).

## Typography & Radius (R-06, R-11)
- **Typeface**: System humanist sans stack (`"Plus Jakarta Sans", "Avenir Next", "Segoe UI", system-ui, sans-serif`) for instant rendering without render-blocking external font requests (R-40).
- **Radius**: Crisp `6px` on inputs/buttons and `10px` on structural containers — avoiding uniform pill shapes.
