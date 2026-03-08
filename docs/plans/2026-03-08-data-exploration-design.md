# Data Exploration Features Design

## Goal

Add five data exploration features to dex for better day-to-day developer experience: row detail view, column sorting, export, server-side pagination, and column hiding.

## Features

### 1. Row Detail View

- `Enter` on results grid opens a modal showing all column:value pairs vertically
- Two-column layout: column name on left, value on right
- j/k to scroll, Esc to close
- Edit trigger moves from `Enter` to `i` or `a` (vim-compatible)

### 2. Column Sorting

- `s` on results grid sorts by the current column ascending (server-side ORDER BY)
- `s` again on same column toggles to descending
- `s` on a different column resets to ascending on that column
- Sort indicator arrow in the column header
- Re-runs the query server-side (works correctly with pagination)

### 3. Export Results

- `:export csv [path]` or `:export json [path]` via command bar
- If path omitted, defaults to `./tablename.csv` or `./tablename.json`
- Exports current result set in memory (not the whole table)
- CSV via `encoding/csv`, JSON as array of objects with column names as keys
- Status bar confirms: "Exported 96 rows to /path/to/file.csv"

### 4. Server-Side Pagination

- Table browsing (sidebar selection) uses `LIMIT 100 OFFSET N`
- `n` = next page, `p` = previous page
- Status bar shows: `page 1/12 | 1-100 of 1,150`
- Total count fetched via `COUNT(*)` on first load
- Page size: 100 rows
- Custom queries (query bar, editor, command bar) run as-is without pagination

### 5. Column Hide/Show

- `c` on results grid toggles hiding the current column
- `C` (shift) shows all columns again
- View-only: data stays in memory, export includes all columns
- Hidden columns tracked as a set of indices, skipped during rendering

## Key Changes

- Results grid `Enter` repurposed from edit to detail view
- Edit triggered by `i` or `a` instead
- `n`/`p` keys repurposed from next/prev page (already existed in keymap but used differently)
- `s`, `c`, `C` are new keybindings on results grid
- Pagination state (offset, total count, sort column/direction) tracked in results model or app model
- Source table name already tracked in results model (used by edit feature)
