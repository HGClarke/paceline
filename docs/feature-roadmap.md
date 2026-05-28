# PacelineCLI — Feature Roadmap

> A product-led prioritization of missing features, with proposed CLI commands, impact rationale, and effort estimates.

**Current capabilities (baseline):** `import`, `rides` (list/filter/TUI), `ride <id>` (detail), `ride <id> stream` (ASCII chart), `stats` (aggregate totals), `delete`. JSON output on all commands.

---

## Priority Matrix

| # | Feature | Priority | Effort | Status |
|---|---------|----------|--------|--------|
| 1 | [Unit preferences (metric / imperial)](#1-unit-preferences-metric--imperial) | 🔴 High | Low | ✅ Completed |
| 2 | [Personal records](#2-personal-records) | 🔴 High | Low | ⬜ Not Started |
| 3 | [Date-range filtering](#3-date-range-filtering) | 🔴 High | Low | ⬜ Not Started |
| 4 | [Ride sorting](#4-ride-sorting) | 🔴 High | Low | ⬜ Not Started |
| 5 | [Richer stats (averages & maximums)](#5-richer-stats-averages--maximums) | 🟡 High | Medium | ⬜ Not Started |
| 6 | [Power curve](#6-power-curve) | 🟡 High | Medium | ⬜ Not Started |
| 7 | [Ride naming & notes](#7-ride-naming--notes) | 🟡 High | Medium | ⬜ Not Started |
| 8 | [Bulk / directory import](#8-bulk--directory-import) | 🟡 High | Medium | ✅ Completed |
| 9 | [HR zone analysis](#9-hr-zone-analysis) | 🟠 Medium | Medium | ⬜ Not Started |
| 10 | [Streak & consistency tracking](#10-streak--consistency-tracking) | 🟠 Medium | Medium | ⬜ Not Started |
| 11 | [Year-over-year comparison](#11-year-over-year-comparison) | 🟠 Medium | Medium | ⬜ Not Started |
| 12 | [Multi-field stream overlay](#12-multi-field-stream-overlay) | 🟠 Medium | Medium | ⬜ Not Started |
| 13 | [AI ride analysis](#13-ai-ride-analysis) | 🟠 Medium | Medium | ⬜ Not Started |
| 14 | [Strava sync](#14-strava-sync) | 🟠 Medium | High | ⬜ Not Started |
| 15 | [Route map in terminal](#15-route-map-in-terminal) | 🔵 Lower | High | ⬜ Not Started |
| 16 | [FTP-based training metrics (NP / IF / TSS)](#16-ftp-based-training-metrics-np--if--tss) | 🔵 Lower | High | ⬜ Not Started |
| 17 | [Watch folder / auto-import](#17-watch-folder--auto-import) | 🔵 Lower | High | ⬜ Not Started |
| 18 | [Homebrew installation](#18-homebrew-installation) | 🔵 Lower | Low | ⬜ Not Started |

---

## 🔴 High Priority, Low Effort

### 1. Unit Preferences (Metric / Imperial)

**Description**
All output currently displays raw SI units — meters, meters-per-second, kilograms. A cyclist in the US or UK sees `48.28 km` as `48280.00 m` and average speed as `6.71 m/s` instead of `15 mph`. The data is correct; the presentation is unusable without mental conversion.

A simple config file at `~/.paceline/config.toml` (or `~/.config/paceline/config.toml`) with a `units` key would control how every `Print*` function formats output. No schema changes required — conversion happens at display time only.

**Why it's impactful**
This is a basic usability gate. Any cyclist outside continental Europe either gives up immediately or does mental arithmetic on every number. Fixing this unlocks the tool for a large portion of its natural audience without touching any core logic.

**Proposed commands**
```bash
# Set units via a dedicated config command
paceline config set units imperial
paceline config set units metric

# View current config
paceline config

# All existing commands automatically respect the setting
paceline rides              # distance now shows miles
paceline stats              # elevation in feet
paceline ride 42            # speed in mph
```

**Config file example**
```toml
# ~/.paceline/config.toml
units = "imperial"   # "metric" (default) | "imperial"
```

---

### 2. Personal Records

**Description**
A `records` command that queries the database for all-time bests: longest single ride by distance, longest by duration, highest average power, highest normalized speed, most elevation in a single ride, and highest average HR. All of this data already lives in the `rides` table — this is purely a `SELECT MAX(...)` query, formatted nicely.

**Why it's impactful**
Cyclists and runners track personal bests obsessively. It's the most emotionally compelling reason to open a training log. Strava's "Achievements" and Garmin's "Personal Records" sections are among their most-viewed pages. Surfacing PRs turns PacelineCLI from a passive log viewer into something users *check* after hard efforts.

**Proposed commands**
```bash
# Show all-time personal records
paceline records

# Example output:
# ┌─────────────────────┬────────────┬──────────────────────────────┐
# │ Record              │ Value      │ Ride                         │
# ├─────────────────────┼────────────┼──────────────────────────────┤
# │ Longest distance    │ 142.3 km   │ 2024-08-10 (ID 88)           │
# │ Longest duration    │ 5h 12m     │ 2024-08-10 (ID 88)           │
# │ Most elevation      │ 2,840 m    │ 2023-07-04 (ID 31)           │
# │ Highest avg power   │ 287 W      │ 2025-03-15 (ID 201)          │
# │ Highest avg speed   │ 38.4 km/h  │ 2024-05-01 (ID 55)           │
# │ Highest avg HR      │ 172 bpm    │ 2025-01-18 (ID 175)          │
# └─────────────────────┴────────────┴──────────────────────────────┘

# JSON output
paceline records --json
```

---

### 3. Date-Range Filtering

**Description**
The `rides` and `stats` commands support filtering by year, month, or exact date — but not by an arbitrary range. A user who wants "everything from my training block" (e.g. Jan 1 – Mar 15) has no way to express that today. Adding `--from` and `--to` flags, each accepting `YYYY-MM-DD`, fills this gap with minimal code — it's just two additional `WHERE` clauses in `buildRideWhere` and `buildStatsWhere`.

**Why it's impactful**
Training is structured around blocks, not calendar months. Users plan around races, rest periods, and coach-assigned phases that don't align with month boundaries. Without range filtering, `stats` is only useful for calendar-aligned reviews. With it, users can answer "how many kilometers did I ride in my 8-week build?" in one command.

**Proposed commands**
```bash
# Filter rides by date range
paceline rides --from 2025-01-01 --to 2025-03-31

# Filter stats by date range
paceline stats --from 2025-01-01 --to 2025-03-31

# Combine with existing flags
paceline rides --from 2025-06-01 --to 2025-06-30 --limit 50

# Shorthand keywords (nice-to-have, lower priority)
paceline stats --range last-30-days
paceline stats --range last-week
paceline stats --range ytd
```

---

### 4. Ride Sorting

**Description**
`rides` always returns results sorted by `recorded_at DESC` (most recent first). There is no way to surface your longest, hardest, or highest-elevation rides without manually scrolling through history. A `--sort` flag mapping to an `ORDER BY` clause is a one-line SQL change backed by a new flag.

**Why it's impactful**
Cyclists frequently ask "what's my longest ride this year?" or "which ride had the most climbing?" These are discovery queries — they don't know the ID, just a property. Without sorting, the answer requires piping to `--json` and writing an external query. Sorting makes the tool answer natural questions directly.

**Proposed commands**
```bash
# Sort by different fields (descending by default for all numeric fields)
paceline rides --sort distance
paceline rides --sort duration
paceline rides --sort elevation
paceline rides --sort power       # avg_power_w
paceline rides --sort speed       # avg_speed_mps
paceline rides --sort date        # (default, same as current behavior)

# Combine with direction
paceline rides --sort distance --order asc
paceline rides --sort distance --order desc   # default

# Combine with other filters
paceline rides --year 2025 --sort elevation --limit 5
```

---

## 🟡 High Priority, Medium Effort

### 5. Richer Stats (Averages & Maximums)

**Description**
The current `stats` command returns four aggregate columns: ride count, total distance, total duration, and total elevation. It omits averages and maximums for the metrics cyclists care most about — power, heart rate, and speed. The data is in the `rides` table (`avg_power_w`, `max_power_w`, `avg_hr_bpm`, `max_hr_bpm`, `avg_speed_mps`, `max_speed_mps`). This is a SQL `AVG()` / `MAX()` addition and a display expansion.

**Why it's impactful**
"How did I perform this month?" is not answered by totals alone. A cyclist wants to know their average power trended up, or that their max HR stayed controlled. Without these metrics, `stats` gives volume context but no quality context. Adding them completes the monthly/weekly review loop that makes a training tool genuinely useful.

**Proposed commands**
```bash
# Existing command, expanded output
paceline stats

# Example expanded output:
# Stats for current month
# ┌──────────────────┬────────────┐
# │ Metric           │ Value      │
# ├──────────────────┼────────────┤
# │ Rides            │ 14         │
# │ Total distance   │ 423.7 km   │
# │ Total duration   │ 18h 42m    │
# │ Total elevation  │ 4,210 m    │
# │ Avg power        │ 231 W      │
# │ Max power        │ 421 W      │
# │ Avg HR           │ 148 bpm    │
# │ Max HR           │ 183 bpm    │
# │ Avg speed        │ 28.4 km/h  │
# └──────────────────┴────────────┘

# Filter as before
paceline stats --year 2025
paceline stats --week 22
paceline stats --from 2025-01-01 --to 2025-03-31  # (once #3 lands)
```

---

### 6. Power Curve

**Description**
A power curve (also called a "mean maximal power" curve) shows the highest average power a rider sustained for each duration: 5 seconds, 1 minute, 5 minutes, 20 minutes, 60 minutes, etc. It is calculated by sliding a window of each duration across the stream data and finding the maximum average. DuckDB's window functions (`AVG(...) OVER (...)`) make this query tractable. The result is displayed as a table of canonical duration → peak power values followed by an ASCII line chart using the same `asciigraph` renderer as `ride <id> stream`.

**Chart rendering note:** the table shows only the 7 canonical intervals, but the chart must be computed at 40–60 logarithmically-spaced durations across the full ride length (not just the 7 table rows). `asciigraph` draws one character column per data point — with only 7 points the chart looks like a staircase of straight-line drops, not a curve. Sampling at ~50 log-spaced durations gives the renderer enough resolution to produce the characteristic smooth exponential decline. The `Width` option stretches the series to fill terminal width, and a custom `XAxisValueFormatter` maps the dense duration values back to human-readable labels (5s, 30s, 1m, 5m, 20m, 60m) at the standard tick positions.

This is the signature analytical feature for any cyclist with a power meter, and the primary reason serious cyclists use platforms like TrainingPeaks, WKO, or Intervals.icu.

**Why it's impactful**
A power curve is how you estimate FTP (Functional Threshold Power), track fitness gains, and understand your physiological strengths (sprinter vs. climber vs. time-trialist). It transforms raw stream data — which is already stored — into the single most informative output you can give a trained cyclist. No other CLI tool provides this natively.

**Proposed commands**
```bash
# Power curve for a single ride (from stream data)
paceline ride 42 power-curve

# Example output:
# Power Curve — Ride 42 (2025-05-10)
# ┌──────────┬─────────┐
# │ Duration │ Power   │
# ├──────────┼─────────┤
# │ 5s       │ 812 W   │
# │ 30s      │ 634 W   │
# │ 1 min    │ 521 W   │
# │ 5 min    │ 380 W   │
# │ 10 min   │ 342 W   │
# │ 20 min   │ 298 W   │
# │ 60 min   │ 261 W   │
# └──────────┴─────────┘
#
# 812 ┤╮
# 754 ┤╰╮
# 695 ┤ ╰─╮
# 637 ┤   ╰╮
# 579 ┤    ╰─╮
# 521 ┤      ╰──╮
# 463 ┤         ╰──╮
# 404 ┤            ╰───╮
# 346 ┤                ╰────╮
# 288 ┤                     ╰──────╮
# 261 ┤                            ╰─────
#      5s  30s  1m       5m    10m  20m  60m
#                    power curve
# (rendered from ~50 log-spaced samples, not just the 7 table rows)

# All-time best power curve across all rides
paceline records power-curve

# Filtered to a period
paceline records power-curve --year 2025
```

---

### 7. Ride Naming & Notes

**Description**
Rides are currently identified only by their source filename (e.g. `2025-05-20T07-32-11.fit`). There is no way to give a ride a human-readable name ("Zwift Race — Watopia Hilly"), tag it by type (road, gravel, indoor, race, group ride), or attach a freeform note ("Felt strong today, new 20-min power PR"). These would be stored in new nullable columns on the `rides` table: `name TEXT`, `note TEXT`, `tags TEXT` (comma-separated or JSON array).

**Why it's impactful**
Without names, a list of rides is an unnavigable list of timestamps. Names and notes are what turn a data log into a *training journal* — the difference between a spreadsheet and a diary. Users who ride consistently will want to annotate significant rides, record how they felt, and distinguish a race from a recovery spin. This is table stakes for any tool that aims to be a primary training record.

**Proposed commands**
```bash
# Set a name for a ride
paceline ride 42 rename "Zwift Race — Watopia Hilly Route"

# Add or replace a note
paceline ride 42 note "Felt strong. New 20-min power PR."

# Clear a note
paceline ride 42 note --clear

# Tag a ride (replaces tags)
paceline ride 42 tag road race group-ride

# Rides list shows name when available (falls back to filename)
paceline rides

# Filter by tag
paceline rides --tag race
paceline rides --tag indoor
```

---

### 8. Bulk / Directory Import

**Description**
The current `import` command accepts a single file path. A user with years of history — downloaded from Garmin Connect, Wahoo, or Zwift — has hundreds or thousands of files in a directory. Making them import one-by-one is a non-starter for onboarding. `import` should accept a directory path and recursively find all `.fit`, `.gpx`, and `.tcx` files, importing each one (skipping duplicates via the existing `ON CONFLICT DO NOTHING` constraint) and printing a summary.

**Why it's impactful**
The first experience with any data tool is loading your existing data. If that experience is painful, users leave before they've seen the tool's value. A single `paceline import ~/Downloads/garmin-export/` that loads 500 rides in one shot is a compelling onboarding moment. The import logic is already idempotent, so the only work is directory traversal and progress reporting.

**Proposed commands**
```bash
# Import a single file (existing behavior)
paceline import activity.fit

# Import all files in a directory
paceline import ~/Downloads/garmin-activities/

# Recursive by default; disable with a flag
paceline import ~/Downloads/ --no-recursive

# Show progress during bulk import
# Importing 847 files...
# [████████████████████░░░░] 412/847  48%  (12 skipped, 0 errors)
#
# Done: 435 imported, 412 skipped (already exists), 0 errors

# Dry-run to preview what would be imported
paceline import ~/Downloads/garmin-activities/ --dry-run
```

---

## 🟠 Medium Priority, Medium Effort

### 9. HR Zone Analysis

**Description**
Heart rate zones divide effort into five training bands based on a percentage of max HR (Zone 1: recovery, Zone 2: aerobic base, Zone 3: tempo, Zone 4: threshold, Zone 5: VO2 max). Given a user-configured `hr_max`, the stream data can be bucketed per-second into zones and summarized as time-in-zone. This requires a `hr_max` config entry and a query over the `streams` table.

**Why it's impactful**
Most structured training plans are written in terms of HR zones. A coach prescribing "2 hours in Zone 2" needs to verify compliance — and without zone analysis, a rider can only guess. Zone distribution is also a quick health check on training balance (too much Zone 3 is the classic "junk miles" trap). This bridges the gap between data collection and training structure.

**Proposed commands**
```bash
# Set max HR in config
paceline config set hr_max 185

# Zone breakdown for a single ride
paceline ride 42 zones

# Example output:
# HR Zones — Ride 42 (2025-05-10)  [Max HR: 185 bpm]
# ┌────────┬───────────────┬──────────┬───────────┬────────┐
# │ Zone   │ Range         │ Time     │ % of ride │ Bar    │
# ├────────┼───────────────┼──────────┼───────────┼────────┤
# │ Z1     │ < 93 bpm      │  8m 12s  │    7%     │ ██     │
# │ Z2     │ 93–111 bpm    │ 42m 30s  │   37%     │ ██████ │
# │ Z3     │ 111–130 bpm   │ 28m 15s  │   25%     │ ████   │
# │ Z4     │ 130–148 bpm   │ 22m 10s  │   19%     │ ███    │
# │ Z5     │ > 148 bpm     │ 13m 53s  │   12%     │ ██     │
# └────────┴───────────────┴──────────┴───────────┴────────┘

# Zone summary across a period
paceline stats --year 2025 --zones
```

---

### 10. Streak & Consistency Tracking

**Description**
A `streaks` command reports training consistency: current active streak (consecutive days with at least one ride), longest-ever streak, and a rolling summary (e.g. "16 of the last 30 days"). Calculated entirely from `recorded_at` timestamps in the `rides` table with a gap-detection query.

**Why it's impactful**
Consistency is the primary driver of athletic improvement — more so than any single hard session. Streak tracking is a powerful behavioral motivator; it's why Duolingo built its entire retention strategy around it. Cyclists and runners who see "you've ridden 18 days in a row" are strongly motivated not to break the chain. This is a low-cost feature that drives daily engagement with the tool.

**Proposed commands**
```bash
paceline streaks

# Example output:
# ┌──────────────────────┬────────────────────────┐
# │ Metric               │ Value                  │
# ├──────────────────────┼────────────────────────┤
# │ Current streak       │ 12 days 🔥             │
# │ Longest streak       │ 31 days (Jan–Feb 2024) │
# │ Last 30 days         │ 22 / 30 days           │
# │ Last 7 days          │ 5 / 7 days             │
# │ This year            │ 87 / 145 days          │
# └──────────────────────┴────────────────────────┘

# JSON for scripting / status bar widgets
paceline streaks --json
```

---

### 11. Year-over-Year Comparison

**Description**
`stats --compare <year>` runs two parallel aggregate queries and prints them side-by-side, with a delta column showing absolute and percentage change. This requires no new data and minimal schema work — it's a presentation layer addition over the existing `GetStats` function.

**Why it's impactful**
Training progression is only visible with comparison. "I rode 4,200 km last year; am I ahead or behind this year?" is the foundational question every serious athlete asks in January. Without YoY comparison, `stats` shows you snapshots but not trajectory. This single feature makes the tool useful for annual goal tracking.

**Proposed commands**
```bash
# Compare current year to last year
paceline stats --year 2025 --compare 2024

# Example output:
# ┌──────────────────┬───────────┬───────────┬──────────────┐
# │ Metric           │ 2025      │ 2024      │ Δ            │
# ├──────────────────┼───────────┼───────────┼──────────────┤
# │ Rides            │ 87        │ 112       │ -25 (-22%)   │
# │ Distance         │ 2,341 km  │ 4,107 km  │ -1,766 (-43%)│
# │ Duration         │ 96h 14m   │ 178h 02m  │ -82h (-46%)  │
# │ Elevation        │ 21,840 m  │ 38,920 m  │ -17,080 (-44%│
# └──────────────────┴───────────┴───────────┴──────────────┘

# Compare any two years
paceline stats --year 2024 --compare 2023

# Compare current month to same month last year
paceline stats --month 5 --year 2025 --compare 2024
```

---

### 12. Multi-Field Stream Overlay

**Description**
`ride <id> stream` currently charts a single field at a time. `asciigraph` supports multiple series on one chart with distinct characters. Allowing `--field power --field hr` together would render both on the same time axis, letting the rider see how HR responds to power output — the fundamental signal of cardiovascular fitness and fatigue.

**Why it's impactful**
Single-field charts answer "what did my power look like?" Multi-field overlays answer "how did my body respond to my power?" — a qualitatively richer question. Seeing HR rise through a hard interval while power holds steady, or watching HR drift upward while power drops on a long ride (cardiac drift), is actionable training insight. This turns the chart from a playback tool into an analysis tool.

**Proposed commands**
```bash
# Single field (existing behavior unchanged)
paceline ride 42 stream --field power

# Overlay two fields on one chart
paceline ride 42 stream --field power --field hr

# Three fields
paceline ride 42 stream --field power --field hr --field cadence

# All fields
paceline ride 42 stream --field all
```

---

### 13. AI Ride Analysis

**Description**
A `ride <n> analyze` subcommand that sends the ride's metrics to the Claude API and returns a natural-language summary of the effort. The prompt includes all available ride data — distance, duration, elevation, average and max values for speed, HR, power, and cadence — and asks for a concise coaching-style read on the ride: how hard it was, what stood out, and any notable patterns. An optional `--verbose` flag triggers stream-level aggregation (e.g. power distribution across 10-minute segments, HR drift over the ride) for a richer analysis.

The API key is configured via the `ANTHROPIC_API_KEY` environment variable, consistent with standard Anthropic SDK usage. A config entry (`paceline config set anthropic_api_key sk-ant-...`) can be added once the config system (prerequisite from Feature 1) lands. Uses the latest Claude model; model can be overridden with `--model`.

**Why it's impactful**
Raw numbers tell you *what* happened; analysis tells you *what it means*. A cyclist who rode 80 km with 1,400 m of elevation and an average HR of 158 bpm might not know if that was a solid aerobic effort or an overreach — but a well-prompted AI can give a calibrated read in seconds. This turns PacelineCLI from a data viewer into a feedback loop, closing the gap with expensive coaching apps.

**Proposed commands**
```bash
# Basic analysis using ride summary metrics
paceline ride 3 analyze

# Example output:
# Analysis — Ride #3 (2025-05-20)
#
# This was a solid aerobic effort — 78.4 km over 2h 47m with 1,420 m of
# climbing puts it firmly in the "long endurance" category. Your average
# power of 198 W at an average HR of 152 bpm suggests you stayed well
# below threshold for most of the ride, consistent with a base-building
# or recovery-from-hard-week context.
#
# The elevation-to-distance ratio (18 m/km) is moderately hilly. Average
# speed of 28.1 km/h is reasonable given the terrain.
#
# One flag: max HR of 181 bpm alongside an average of 152 suggests at
# least one hard spike — likely a short climb or sprint. Worth noting if
# this was meant to be a purely easy day.

# Include stream-level detail in the prompt (slower, richer)
paceline ride 3 analyze --verbose

# JSON output (analysis text as a field)
paceline ride 3 analyze --json

# Override model
paceline ride 3 analyze --model claude-opus-4-7
```

**Implementation notes**
- Requires `ANTHROPIC_API_KEY` env var (or future config entry)
- v1 prompt uses only `rides` table columns — no stream queries needed
- `--verbose` adds stream aggregates: per-10min power/HR averages, and total time above threshold HR (if HR data exists)
- Gracefully degrades when optional metrics (power, HR) are absent — prompt omits unavailable fields
- Displays a spinner while waiting for the API response

---

### 14. Strava Sync

**Description**
A `sync strava` command that authenticates with Strava via OAuth 2.0 and pulls all activities into the local database. On first run it fetches the full activity history; on subsequent runs it fetches only new activities (those recorded after the most recent locally-stored ride). Activity IDs from Strava are stored in a `strava_activity_id` column on the `rides` table, making re-syncs fully idempotent — importing an already-stored activity is a no-op. Ride detail fields (distance, duration, elevation, HR, power, speed, cadence) are mapped from Strava's API response to PacelineCLI's `rides` schema. Stream data (second-by-second power, HR, etc.) is fetched lazily per-ride when `ride <n> stream` is first requested, to avoid exhausting Strava's rate limits on bulk sync.

For v1, only Strava is supported. The command structure (`sync strava`) is designed to accommodate additional providers in the future via additional subcommands (`sync garmin`, `sync wahoo`), each accepting the same set of flags.

**Why it's impactful**
Most cyclists already record rides on Strava. Requiring manual FIT/GPX export and import is a significant friction point for new users; the majority will never do it more than once. A `sync strava` that pulls the last year of rides in one shot removes onboarding friction entirely and makes PacelineCLI a live mirror of a Strava account. Idempotency means users can sync on a schedule without thinking about it.

**OAuth and authentication**
Strava uses OAuth 2.0 with an authorization code flow. For a CLI tool, the standard pattern is:
1. Open the user's browser to Strava's authorization URL
2. Start a short-lived local HTTP listener on a redirect URI (`http://localhost:PORT/callback`)
3. Exchange the returned code for access + refresh tokens
4. Persist tokens in `~/.paceline/strava-token.json`; refresh silently on expiry

**Rate limits**
Strava enforces 100 requests per 15 minutes and 1,000 per day. Sync handles this with:
- **Incremental fetches**: subsequent syncs only request activities after `last_sync_at`, keeping request counts small
- **Rate-limit backoff**: on a 429 response, read `X-RateLimit-Reset` and sleep until the window resets, then continue — the user sees a paused progress bar, not a failure
- **Lazy stream fetching**: stream data is fetched per-ride on demand, not during bulk sync, spreading requests across sessions. The activity list endpoint returns up to 200 activities per page (~5 requests for 1,000 activities); fetching streams costs 1 request per ride. Deferring streams keeps the initial sync well within rate limits.

**Idempotency and resumability**
A new nullable `strava_activity_id TEXT UNIQUE` column on `rides` carries the Strava activity ID. Each activity is written to the DB immediately as it is fetched — not batched at the end. This makes the sync inherently resumable without any explicit checkpoint state: **the DB is the checkpoint**.

If a sync is interrupted (network error, process killed, daily rate limit hit):
1. On retry, re-fetch the Strava activity list (cheap — just IDs and summary fields, ~3–5 requests regardless of total count)
2. Filter out activity IDs already present in the DB
3. Fetch details only for the remainder

The retry costs the same as completing the original sync — no duplicate API calls, no duplicate DB writes. `ON CONFLICT (strava_activity_id) DO NOTHING` ensures that even if an activity is sent twice, the insert is a no-op.

The `last_sync_at` watermark serves a separate purpose: it optimises *subsequent incremental syncs* by passing `after=last_sync_at` to Strava so only new activities are fetched. It is only updated after a sync completes fully. An interrupted sync leaves the watermark unchanged, so the retry re-walks from the same starting point and relies on the DB-as-checkpoint to skip already-imported activities efficiently.

| Scenario | Behaviour |
|---|---|
| Initial sync interrupted at activity 500/847 | Retry re-fetches list, skips 500 already stored, fetches remaining 347 |
| Daily rate limit hit mid-sync | Wait until midnight UTC reset; retry resumes from where it stopped |
| Incremental sync interrupted | `last_sync_at` not updated; retry re-fetches since the previous watermark, skips duplicates |
| Activity imported twice (any cause) | Second insert is a no-op — no duplicate row created |

File-based rides (imported via `import`) have a `NULL` `strava_activity_id`; the two import paths coexist without conflict.

**Proposed commands**
```bash
# First run: authenticate and full sync
paceline sync strava

# Output during first sync:
# Opening browser for Strava authorization…
# Authorized. Fetching activities…
# [████████████████████░░░░] 312/847  37%
# Done: 847 activities synced, 0 skipped.

# Subsequent runs: incremental only
paceline sync strava
# Fetching activities since 2025-05-18…
# Done: 3 new activities synced, 0 skipped.

# Force a full re-sync (useful after schema changes)
paceline sync strava --full

# Disconnect and remove stored tokens
paceline sync strava --revoke

# Future providers (not v1)
paceline sync garmin
paceline sync wahoo
```

**Implementation notes**
- Schema migration: add `strava_activity_id TEXT UNIQUE` column to `rides`; add a `sync_state` table (or a row in a `meta` table) to store `last_sync_at` (Unix timestamp, updated only on full success)
- Write each activity to the DB immediately on fetch — never accumulate in memory and batch-insert
- OAuth token storage: `~/.paceline/strava-token.json` with `access_token`, `refresh_token`, `expires_at`
- Strava pagination: `GET /v3/athlete/activities?after=<unix>&per_page=200&page=N` — iterate until empty page
- On retry: query `SELECT strava_activity_id FROM rides WHERE strava_activity_id IS NOT NULL` into a set, then skip any fetched activity whose ID is already present before making the detail request
- Field mapping: Strava's `moving_time` → `duration_s`, `distance` → `distance_m`, `total_elevation_gain` → `elevation_gain_m`, `average_watts` → `avg_power_w`, `average_heartrate` → `avg_hr_bpm`, etc.
- Stream fetch: `GET /v3/activities/{id}/streams?keys=time,watts,heartrate,velocity_smooth,cadence,altitude` — deferred to first `ride <n> stream` call, cached in the `streams` table
- Client ID / secret: `STRAVA_CLIENT_ID` and `STRAVA_CLIENT_SECRET` env vars (or future config entry)

---

## 🔵 Lower Priority, Higher Effort

### 15. Route Map in Terminal  

**Description**
Latitude and longitude are already captured in the `streams` table but are never displayed. A `route` subcommand would project the GPS coordinates onto a 2D ASCII canvas — normalizing lat/lon to terminal character positions and drawing the path using Unicode block characters or Braille patterns. Libraries like `go-term-markdown` or custom projection code can accomplish this.

**Why it's impactful**
A route map is the most visually striking output a cycling CLI can produce. It's the feature that gets screenshots shared on social media and shown to friends. Beyond the "wow" factor, it has real utility — confirming a route was recorded correctly, identifying where a ride went, or comparing two routes. Lat/lon data is captured and wasted today; this makes it visible.

**Proposed commands**
```bash
# ASCII route map for a ride
paceline ride 42 route

# Control output size
paceline ride 42 route --width 120 --height 40

# Example output (illustrative):
#     ·····
#   ··     ··
#  ·          ·
#  ·    ↑      ·
#   ··  S    ··
#     ·····
#  (S = start, dots = route path)
```

---

### 16. FTP-Based Training Metrics (NP / IF / TSS)

**Description**
Normalized Power (NP), Intensity Factor (IF), and Training Stress Score (TSS) are the standard currency of power-based training, popularized by TrainingPeaks and used by professional coaches worldwide. They require a user-set FTP (Functional Threshold Power) stored in config. NP is calculated from a rolling 30-second average of the power stream raised to the fourth power. IF = NP / FTP. TSS = (duration × NP × IF) / (FTP × 3600) × 100.

**Why it's impactful**
These metrics are how coaches and athletes quantify training load and prescribe recovery. TSS in particular is the foundation for modeling fatigue (ATL) and fitness (CTL) over time. Any cyclist using a structured training plan already tracks these numbers; if PacelineCLI doesn't provide them, those users stay on TrainingPeaks or Intervals.icu regardless of how good everything else is. Adding NP/IF/TSS turns PacelineCLI from a log into a training management tool.

**Proposed commands**
```bash
# Set FTP in config
paceline config set ftp 265

# Show power metrics for a ride (added to ride detail output when FTP is set)
paceline ride 42

# Example additional rows in ride detail:
# │ Normalized power   │ 251 W      │
# │ Intensity factor   │ 0.95       │
# │ Training stress    │ 98.2 TSS   │

# Include in stats aggregates
paceline stats  # total weekly TSS, chronic training load trend
```

---

### 17. Watch Folder / Auto-Import

**Description**
A `watch` command starts a background daemon that monitors a specified directory for new `.fit`, `.gpx`, and `.tcx` files and automatically imports them. On macOS this would use `FSEvents` via a Go library (`github.com/fsnotify/fsnotify`). The daemon writes a PID file to `~/.paceline/` and can be stopped with `paceline watch stop`.

**Why it's impactful**
Many cyclists sync their Garmin or Wahoo to a desktop app that writes FIT files to a known directory. Auto-import closes the loop: finish a ride, sync device, open terminal — the ride is already there. This elevates PacelineCLI from a tool you have to feed into a tool that stays up to date automatically. It's the difference between a passive archive and a live training dashboard.

**Proposed commands**
```bash
# Start watching a directory (runs in background)
paceline watch ~/Library/Application\ Support/Garmin/Devices/

# Check watch status
paceline watch status

# Stop the background watcher
paceline watch stop

# View watch log
paceline watch log

# Example status output:
# Watching: ~/Library/Application Support/Garmin/Devices/
# Status:   running (PID 48291)
# Uptime:   3 days, 14h 22m
# Imported: 12 files since start
```

---

### 18. Homebrew Installation

**Description**
Allow users to install Paceline via [Homebrew](https://brew.sh/) — the de facto package manager for macOS — with a single command:

```bash
brew install HGClarke/paceline/paceline
```

This requires two things: an automated release pipeline using [GoReleaser](https://goreleaser.com/) that cross-compiles binaries for macOS (arm64 and amd64) and Linux and uploads them as GitHub Release assets; and a [Homebrew tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap) — a separate public GitHub repository (`homebrew-paceline`) containing a Ruby formula file that GoReleaser auto-generates and pushes on each tagged release. The formula points Homebrew at the correct binary for the user's architecture and computes a SHA256 checksum for integrity.

**Why it's impactful**
`go install` requires Go to be installed and `~/go/bin` on PATH — two steps many non-developer cyclists will never complete. Homebrew removes both barriers: it handles PATH automatically and has no Go prerequisite. It's the installation path macOS users expect and trust. Adding it turns the install story from a developer workflow into a one-liner for any cyclist.

**Effort estimate: Low**
The majority of the work is configuration, not code:
- `.goreleaser.yml` — ~30 lines of YAML configuring build targets, archive naming, and checksum generation
- `.github/workflows/release.yml` — GitHub Actions workflow that runs GoReleaser on `git tag v*` pushes (~20 lines)
- `homebrew-paceline` repo — a new public GitHub repository; GoReleaser auto-generates and pushes the formula file on each release, so no manual formula maintenance is needed after initial setup
- One-time setup: create a GitHub personal access token scoped to the tap repo so GoReleaser can push the formula update

Total estimated time: **2–4 hours** for initial setup; subsequent releases are fully automated.

**Prerequisite**
A tagged versioning convention (e.g. `v1.0.0`) and a version injection mechanism in the binary. The version should be injected at build time via ldflags:
```bash
go build -ldflags "-X github.com/HGClarke/paceline/cmd.version={{.Version}}" -o paceline .
```
This also unlocks a `paceline version` command as a natural side effect.

**Proposed commands**
```bash
# One-time tap registration (only needed once per machine)
brew tap HGClarke/paceline

# Install
brew install HGClarke/paceline/paceline

# Upgrade to latest release
brew upgrade paceline

# Or in one shot (tap + install)
brew install HGClarke/paceline/paceline
```

---

## Implementation Notes

### Config System (Prerequisite for Features 1, 9, 13, 16)
Features 1, 9, 13, and 16 all require a persistent config file. This should be implemented first as a shared foundation:

```toml
# ~/.paceline/config.toml
units   = "imperial"   # "metric" | "imperial"
hr_max  = 185          # used for zone calculations
ftp     = 265          # used for NP/IF/TSS
```

A `paceline config` command family (`config set <key> <value>`, `config get <key>`, `config list`) makes this user-accessible without requiring manual file edits.

### Suggested Build Order
Given dependencies and quick wins:

1. **Config system** — unlocks 3 features, low effort
2. **Units** — immediate user-facing win, unblocked by config
3. **Date-range filtering** — pure SQL, no dependencies
4. **Ride sorting** — pure SQL, no dependencies
5. **Personal records** — pure SQL, high emotional value
6. **Richer stats** — SQL + display expansion
7. **Bulk import** — directory traversal + progress display
8. **Ride naming & notes** — schema migration + new commands
9. **Power curve** — stream query with window functions
10. **Multi-field overlay** — display layer only
11. **HR zones** — needs config (hr_max), stream query
12. **Streak tracking** — gap detection query
13. **YoY comparison** — display layer + parallel queries
14. **AI ride analysis** — needs API key (env var or config); v1 uses ride summary only
15. **Strava sync** — OAuth flow, schema migration, incremental fetch
16. **FTP metrics** — needs config (ftp), stream computation
17. **Route map** — GPS projection, standalone
18. **Watch folder** — background daemon, most complex
19. **Homebrew installation** — GoReleaser setup, tap repo, GitHub Actions release workflow
