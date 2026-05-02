# Server-Driven UI Stack - Discussion Notes and Implementation Guide

Date: 2026-05-01

## Overview

This document captures a comprehensive discussion on building modern web applications using a server-driven UI stack as an alternative to JavaScript SPA frameworks (Svelte, React, Next.js, etc.). The goal is to explore whether this approach can deliver a productive, maintainable development experience in 2026.

The document is organized as follows:

1. **The Stack** - what tools we chose and why
2. **How It Works** - the interactivity model, HTMX 4 features, and integration patterns
3. **Comparisons** - how this stack compares to JS frameworks, Hotwire, and Astro
4. **Tradeoffs** - where this stack shines and where it struggles
5. **Implementation** - Go + Chi backend, templ, asset management, developer experience
6. **The PoC Application** - bookmark manager design and project structure
7. **Deployment** - Docker, AWS Lambda, best practices
8. **Testing** - the testing pyramid for server-driven UIs
9. **LLM-Friendliness** - why this stack works well with AI-assisted development

## The Stack

| Layer | Tool | Role |
|-------|------|------|
| Server | Go + Chi | Routing, business logic, HTML rendering |
| Templating | templ (recommended) or Go `html/template` | Type-safe, composable server-side HTML |
| Interactions | HTMX 4 | HTTP requests, DOM swaps |
| Components | Basecoat | UI component library (Tailwind + shadcn-style) |
| Behavior | Stimulus | Client-side JS organization |
| Styling | Tailwind CSS | Utility CSS (via Basecoat) |

Each tool fills a distinct role without overlap:

- **HTMX 4** - replaces the SPA router + data fetching layer. The server returns HTML fragments, skipping the JSON-to-client-render cycle entirely.
- **Basecoat** - provides a component library (Tailwind + shadcn-style) that works with server-rendered HTML. No React/Svelte needed just for UI components.
- **Stimulus** - provides a thin JS interactivity layer (toggling classes, managing small state, connecting DOM elements to behavior) without a virtual DOM or reactivity system.
- **Go + Chi** - minimal, composable HTTP server. Just a router and middleware on top of `net/http`.
- **templ** - type-safe HTML templating for Go with LSP support, compile-time error checking, and first-class HTMX integration. Recommended over `html/template`.

## HTMX 4 - Key Features

HTMX 4 is a major release (site: https://four.htmx.org/). Notable changes relevant to this stack:

### Built-in Morphing

`innerMorph` / `outerMorph` swap styles using the idiomorph algorithm are baked in. Morph-based swaps preserve DOM state (focus, scroll position, form input values) across updates - closing a gap where SPAs previously had a clear UX advantage.

### `<hx-partial>` for Multi-Target Updates

Target multiple elements from a single response without the OOB swap workaround:

```html
<hx-partial hx-target="#messages" hx-swap="beforeend">
    <div>New message</div>
</hx-partial>

<hx-partial hx-target="#count">
    <span>5</span>
</hx-partial>
```

### `hx-status` for Per-Status-Code Behavior

Handle validation errors differently from server errors at the HTML attribute level:

```html
<form hx-post="/save"
      hx-status:422="swap:innerHTML target:#errors select:#validation-errors"
      hx-status:5xx="swap:none push:false">
</form>
```

### Other Notable Changes

- **`fetch()` replaces XHR** - modern API, better streaming support.
- **Explicit inheritance** - attributes no longer inherit implicitly down the DOM. Opt in with `:inherited` suffix.
- **Optimistic UI extension** - shows expected content from a template before the server responds, addressing perceived latency.
- **Error responses swap by default** - 4xx and 5xx responses are now swapped. Only 204 and 304 do not swap.
- **60-second default timeout** - previously no timeout.
- **Preload extension** - prefetch on hover for near-instant page loads.

### Migration from HTMX 2.x

- Run the upgrade checker: `npx htmx.org@next upgrade-check -- ./path/to/project`
- Rename `hx-disable` to `hx-ignore`, then `hx-disabled-elt` to `hx-disable`
- Event names follow new pattern: `htmx:phase:action[:sub-action]`
- All error events consolidated to `htmx:error`
- Removed JS API methods replaced by native JS equivalents

**IMPORTANT**: Always refer to https://four.htmx.org/docs for the latest HTMX 4 APIs and patterns.

## Strengths vs JS Frameworks (SvelteKit, Next.js, etc.)

These are the areas where the server-driven approach has a clear advantage:

- **Simplicity** - no build step required for the frontend (or a minimal one for Tailwind). No hydration, no client-side routing bugs, no bundle size anxiety.
- **Backend-agnostic** - works with any server that returns HTML. Not locked into a JS runtime on the server.
- **Fewer failure modes** - no hydration mismatches, no client/server state sync issues, no "loading skeleton to content flash" patterns.
- **Smaller surface area** - less JS shipped to the client, fewer dependencies, simpler mental model.
- **Single binary deployment** - `go build` gives you one file. No Node runtime, no `node_modules`.

## Tradeoffs and Limitations

### Where This Stack Gets Harder

- **Rich interactivity** - anything needing complex client-side state (drag-and-drop builders, real-time collaborative editing, complex form wizards with lots of conditional logic) will fight this model. Stimulus handles simple cases but has a ceiling.
- **Perceived performance** - every interaction round-trips to the server. HTMX mitigates this with swap transitions, optimistic UI, and preload, but on high-latency connections a SPA will feel snappier for navigation-heavy UIs.
- **Ecosystem** - the JS framework ecosystem has massive component libraries, form validation, animation, state machines, etc. With this stack, you build more yourself or rely on smaller communities.
- **Fine-grained reactivity** - Svelte's runes and derived values handle highly interconnected reactive UIs elegantly. In this stack, each update is a separate HTMX request or `<hx-partial>` multi-target response. Works, but less elegant for deeply interdependent state.

### Chattiness Concern

This stack is chattier than a SPA since every interaction round-trips to the server. Mitigation strategies:

- **Debounce**: `hx-trigger="input changed delay:300ms"`
- **Minimum input length**: `hx-trigger="input changed delay:300ms[this.value.length >= 2]"`
- **Cancel in-flight requests**: `hx-sync="this:replace"`
- **HTTP caching**: server returns `Cache-Control` headers, browser skips duplicate requests
- **Preload extension**: prefetch on hover
- **Optimistic UI extension**: show expected result immediately

In practice, a Go server returning a filtered HTML fragment is faster than most SPA apps doing JSON parse, state update, virtual DOM diff, and re-render. The chattiness is rarely the bottleneck.

### When to Reach for a JS Framework Instead

- Charts/visualizations (use as islands - see below)
- Real-time collaborative editing
- Complex drag-and-drop interfaces
- Offline-first apps
- Rich media editors (video, image, audio)

Even for these, you can embed them as isolated widgets within a server-rendered page. You don't need the entire app to be a SPA just because one page has a chart.

## Interactivity Model - Svelte vs Hypermedia

### The Mental Model Shift

In Svelte, state lives on the client and the UI reacts to it. In this stack, state lives on the server and the UI reacts to HTML responses.

### Svelte Primitives Mapped to Hypermedia Equivalents

| Svelte | Hypermedia Stack Equivalent |
|--------|---------------------------|
| `$state` | Server-side data (DB, session) |
| `$derived` | Server computes it during rendering |
| `$effect` | HTMX events + Stimulus callbacks |
| `bind:value` | Standard form inputs, HTMX sends them |
| Reactive re-render | HTMX swap with new HTML from server |
| Client-side stores | Not needed - server is the store |

### Three Tiers of Interactivity

**Tier 1: Pure HTMX (no JS at all) - covers ~70% of interactions**

```html
<!-- Toggle bookmark favorite -->
<button hx-patch="/bookmarks/42/toggle-fav" hx-swap="outerHTML">
  Favorite
</button>
<!-- Server returns the same button with updated state -->
```

**Tier 2: Stimulus for UI-only behavior - covers ~25%**

Things that don't need the server: showing/hiding elements, managing focus, keyboard shortcuts, clipboard operations, form field interactions.

```javascript
// tag_input_controller.js
import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  static targets = ["input", "list", "hidden"]

  add(e) {
    if (e.key !== "Enter") return
    const tag = this.inputTarget.value.trim()
    if (!tag) return

    this.listTarget.insertAdjacentHTML("beforeend",
      `<span data-bc="badge" data-action="click->tag-input#remove">${tag} x</span>`)

    this.hiddenTarget.insertAdjacentHTML("beforeend",
      `<input type="hidden" name="tag" value="${tag}">`)

    this.inputTarget.value = ""
  }

  remove(e) {
    const tag = e.target.textContent.slice(0, -2).trim()
    e.target.remove()
    this.hiddenTarget.querySelector(`[value="${tag}"]`).remove()
  }
}
```

**Tier 3: Stimulus + HTMX coordinated - covers ~5%**

When client behavior needs to trigger server updates or react to them:

```html
<div data-controller="shortcuts"
     data-action="keydown.ctrl+k@document->shortcuts#focusSearch">

  <input name="search" data-shortcuts-target="searchInput"
         hx-get="/bookmarks"
         hx-trigger="input changed delay:300ms"
         hx-target="#bookmark-list">
</div>
```

## Stimulus vs Alternatives

### Why Stimulus Over Alpine.js

- **Alpine UI components are a paid product**. Without it, Alpine gives you reactivity primitives but no pre-built components.
- **Basecoat already provides the component library** (open source, shadcn-style) - buttons, dialogs, selects, tabs, etc.
- **Stimulus keeps JS in separate controller files**, which scales better organizationally. Alpine scatters JS logic across HTML attributes.
- With HTMX attributes + Basecoat attributes + Alpine attributes, you'd have three different attribute systems competing in markup. Stimulus avoids this.

### Why Not jQuery/Umbrella.js

jQuery/Umbrella.js are DOM manipulation libraries. Stimulus is an organizational framework. Different problems.

Modern browser APIs have closed the convenience gap:
- `document.querySelector` / `querySelectorAll` replaced Sizzle
- `element.closest()`, `.matches()`, `.classList`, `.remove()` are all native
- `fetch()` replaced `$.ajax`

In this stack, HTMX handles DOM mutations and Basecoat handles component styling. What remains is behavioral wiring - exactly what Stimulus is designed for. A DOM library would be redundant.

### Other Alternatives Considered

Stimulus gives similar vibes to Backbone.js back in the day - both give you structure and conventions without a rendering engine. Backbone had models/collections/views, Stimulus has controllers/targets/actions. Same philosophy: organize your JS, don't take over the DOM.

Beyond Alpine, the full landscape:

| Tool | Vibe | Tradeoff |
|------|------|----------|
| Stimulus | Backbone-like structure, explicit | More boilerplate, clear organization |
| Alpine.js | Vue-like inline reactivity | Logic in HTML, paid component library |
| Datastar | HTMX + Alpine in one | Newer, smaller community, replaces HTMX too |
| Lit | Web components standard | More ceremony, but real encapsulation |
| Vanilla + conventions | Zero dependencies | You maintain the pattern yourself |

- **Hyperscript** - from the HTMX team. Inline scripting language in HTML attributes (`_="on click toggle .active"`). Custom syntax is harder to debug.
- **Datastar** - a newer entrant explicitly designed as an HTMX alternative. Combines what HTMX and Alpine do into a single library using `data-*` attributes for both server communication and client-side reactivity. Smaller community but interesting if you want one tool instead of HTMX + Stimulus. Worth watching.
- **mini.js** - very minimal (~2KB), uses `data-*` attributes for simple bindings and event handling. Closer to Alpine in spirit but much smaller. Less structure than Stimulus.
- **Lit (web components)** - Google's library for building web components. Different paradigm - you create `<bookmark-card>` custom elements with encapsulated behavior. Works with server-rendered HTML since web components can progressively enhance. More boilerplate than Stimulus but gives real encapsulation and Shadow DOM.
- **Petite-Vue** - Vue's minimal distribution (~6KB). Less community momentum.
- **Vanilla JS with `data-*` attributes** - works for small apps but you end up reinventing Stimulus's controller/target/action pattern.

Stimulus and Alpine are the two mature, well-documented options with real communities. Everything else is either too new (Datastar), too minimal (mini.js), or a different paradigm (Lit). For a server-driven stack where JS is the minority of the codebase, Stimulus's "boring" structured approach is exactly what you want.

## Astro Comparison

Astro sits in the middle ground - server-first like this stack but still in the JS framework world.

| Concern | Astro | Go + HTMX Stack |
|---------|-------|-----------------|
| Server runtime | Node.js (required for SSR) | Go binary |
| Build step | Vite (required) | Tailwind CLI (optional esbuild) |
| Templating | `.astro` files (JSX-like) | Go `html/template` |
| Islands | Built-in (`client:load`, `client:visible`) | Manual via Stimulus controllers |
| Component library | Any (React, Svelte, Vue) | Basecoat |
| Deployment | Node server or edge runtime | Single binary |
| Dependencies | Hundreds (`node_modules`) | Go modules + 3 JS files |

Astro provides conveniences (automatic code splitting, image optimization, content collections, file-based routing) but requires the Node ecosystem. If the goal is to leave JS framework complexity entirely, Astro is a half-measure. The Go + HTMX stack is the full commitment.

## Hotwire (Turbo + Stimulus) and the Rails Connection

Hotwire (Turbo + Stimulus) is the default frontend approach in Rails 7+. It follows the exact same philosophy - server renders HTML, minimal client-side JS, partial page updates over the wire. The Rails community has been doing this for years.

### Is Hotwire Tied to Rails?

Technically no, practically somewhat.

**Stimulus** is fully framework-agnostic. It's just a JS library that reads `data-*` attributes from the DOM. It doesn't care what rendered the HTML. Works perfectly with Go, Python, PHP, anything.

**Turbo** is where the coupling creeps in:

- **Turbo Drive** (full-page navigation) - framework-agnostic. Intercepts link clicks and form submissions, fetches HTML, swaps the `<body>`.
- **Turbo Frames** (partial updates) - mostly agnostic, but expects specific `<turbo-frame>` HTML elements in responses.
- **Turbo Streams** (multi-target updates via WebSocket/SSE) - strongest Rails coupling. Rails has `turbo-rails` gem with helpers that auto-generate Turbo Stream responses, automatic broadcasts from ActiveRecord model callbacks, ActionCable integration. Without Rails, you hand-craft the `<turbo-stream>` XML format yourself.

### Hotwire vs This Stack

| Concern | Hotwire (Turbo + Stimulus) | This Stack (HTMX 4 + Stimulus) |
|---------|---------------------------|-------------------------------|
| Partial updates | Turbo Frames (`<turbo-frame>`) | HTMX (`hx-get`, `hx-target`, `hx-swap`) |
| Multi-target updates | Turbo Streams (`<turbo-stream>`) | `<hx-partial>` or `hx-swap-oob` |
| Full-page navigation | Turbo Drive | `hx-boost` |
| Client-side behavior | Stimulus | Stimulus (same) |
| Server coupling | Strong Rails conventions | None - any server returning HTML |
| Attribute style | Custom HTML elements (`<turbo-frame>`) | `hx-*` attributes on any element |
| Real-time | Turbo Streams over WebSocket/SSE | HTMX SSE/WS extensions |

Key difference: Turbo uses custom HTML elements while HTMX uses attributes on standard HTML elements. HTMX's approach is more flexible - any element can trigger any request with any swap strategy. Turbo is more opinionated and structured.

### Why HTMX Is the Better Choice for Go

- **No framework assumptions** - HTMX doesn't expect any specific response format. Return HTML, done. Turbo expects `<turbo-frame>` wrappers and `<turbo-stream>` action elements.
- **Simpler mental model** - HTMX attributes directly on elements vs wrapping content in Turbo Frame elements and managing frame IDs.
- **Larger non-Rails community** - HTMX's community spans Go, Python, PHP, Rust, Java. Most Turbo content assumes Rails.
- **HTMX 4 closed the gaps** - `<hx-partial>` gives Turbo Streams-like multi-target updates without the custom element format. Morph swaps give Turbo's morphing. Optimistic UI goes beyond what Turbo offers out of the box.

You could use Turbo + Stimulus with Go, but you'd be fighting upstream - the docs, examples, and community all assume Rails. This stack is essentially "Hotwire for the rest of us" - same philosophy, better fit for Go.

## Visualization / Chart Integration

Charts are client-side JS widgets - no way around that. The pattern is to use Stimulus controllers as adapters between server-provided data and the charting library.

### Basic Pattern

```html
<!-- Server returns this via HTMX -->
<div data-controller="chart"
     data-chart-type-value="bar"
     data-chart-config-value='{"labels":["Mon","Tue","Wed"],"data":[5,12,8]}'>
  <canvas data-chart-target="canvas"></canvas>
</div>
```

```javascript
// chart_controller.js
import { Controller } from "@hotwired/stimulus"
import Chart from "chart.js/auto"

export default class extends Controller {
  static targets = ["canvas"]
  static values = { type: String, config: Object }

  connect() {
    this.chart = new Chart(this.canvasTarget, {
      type: this.typeValue,
      data: this.configValue
    })
  }

  disconnect() {
    this.chart.destroy()
  }
}
```

### Animated Updates with Morph Swaps

Use HTMX's morph swap so the element is updated in-place, and Stimulus detects the value change:

```html
<select hx-get="/stats/chart" hx-target="#chart-container" hx-swap="innerMorph">
```

```javascript
export default class extends Controller {
  static targets = ["canvas"]
  static values = { type: String, config: Object }

  connect() {
    this.chart = new Chart(this.canvasTarget, {
      type: this.typeValue,
      data: this.configValue
    })
  }

  configValueChanged() {
    if (!this.chart) return
    this.chart.data = this.configValue
    this.chart.update("active") // animated transition
  }

  disconnect() { this.chart.destroy() }
}
```

Works with any charting library: Chart.js, D3.js, Apache ECharts, Plotly, Observable Plot.

## Go + Chi Backend

### Why Go + Chi

- **Single binary deployment** - `go build` gives one file. No Node runtime, no `node_modules`.
- **`html/template` in the standard library** - fast, auto-escapes output.
- **Chi is minimal** - just a router and middleware on `net/http`. No ORM, no magic.
- **Performance** - starts in milliseconds, handles thousands of concurrent requests with minimal memory. Low latency per request matters when every interaction is a server round-trip.

### Basic Server Structure

```go
package main

import (
    "html/template"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

var tmpl = template.Must(template.ParseGlob("templates/*.html"))

func main() {
    r := chi.NewRouter()
    r.Use(middleware.Logger)

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        tmpl.ExecuteTemplate(w, "index.html", nil)
    })

    // HTMX partial - returns an HTML fragment, not a full page
    r.Post("/tasks", func(w http.ResponseWriter, r *http.Request) {
        r.ParseForm()
        task := Task{Title: r.FormValue("title"), Priority: r.FormValue("priority")}
        // save task...
        tmpl.ExecuteTemplate(w, "task-item", task)
    })

    http.ListenAndServe(":3000", r)
}
```

### Full vs Partial Response Middleware

```go
func fullOrPartial(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("HX-Request") == "true" {
            r = r.WithContext(context.WithValue(r.Context(), "partial", true))
        }
        next.ServeHTTP(w, r)
    })
}
```

### Template Fragment Example

```html
{{define "task-item"}}
<li data-bc="list-item">
  <span>{{.Title}}</span>
  <span data-bc="badge">{{.Priority}}</span>
  <button data-bc="button" variant="ghost"
          hx-delete="/tasks/{{.ID}}" hx-target="closest li" hx-swap="outerHTML">
    Delete
  </button>
</li>
{{end}}
```

## Templ - Type-Safe Templating (Recommended)

templ (https://templ.guide/) is a type-safe HTML templating language for Go. It compiles `.templ` files into Go code, giving you compile-time error checking, IDE autocompletion via LSP, and composable components as Go functions. It has first-class support for both Chi and HTMX.

### Why templ Over `html/template`

| Concern | `html/template` | templ |
|---------|-----------------|-------|
| Type safety | None - runtime errors for missing data | Compile-time - wrong type = build fails |
| IDE support | Minimal | Full LSP - autocomplete, go-to-definition, rename |
| Composition | `{{template "name" .}}` (stringly-typed) | `@Component(args)` (function calls) |
| Go logic | Limited template functions | Full Go - `if`, `for`, `switch`, function calls |
| Refactoring | Find-and-replace strings | Standard Go refactoring tools |
| Error messages | Runtime, often cryptic | Compile-time, points to the `.templ` file and line |

### How It Works with Chi

Every templ component implements `templ.Component` with a `Render(ctx context.Context, w io.Writer)` method. Since `http.ResponseWriter` implements `io.Writer`, it plugs directly into Chi handlers:

```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/a-h/templ"
)

func main() {
    r := chi.NewRouter()

    // Static page - use templ.Handler directly
    r.Get("/about", templ.Handler(aboutPage()).ServeHTTP)

    // Dynamic data - call Render in the handler
    r.Get("/bookmarks", func(w http.ResponseWriter, r *http.Request) {
        bookmarks, _ := db.GetAll()
        components.BookmarkList(bookmarks).Render(r.Context(), w)
    })

    // HTMX partial - same pattern, just render a fragment component
    r.Get("/bookmarks/filter", func(w http.ResponseWriter, r *http.Request) {
        tags := r.URL.Query()["tag"]
        bookmarks, _ := db.GetBookmarksByTags(tags)
        components.BookmarkListItems(bookmarks).Render(r.Context(), w)
    })

    http.ListenAndServe(":3000", r)
}
```

### templ Component Files

```templ
// components/bookmarks.templ
package components

import "fmt"

templ BookmarkList(bookmarks []Bookmark) {
    <ul id="bookmark-list" data-bc="list">
        for _, b := range bookmarks {
            @BookmarkItem(b)
        }
    </ul>
}

templ BookmarkItem(b Bookmark) {
    <li data-bc="card">
        <a href={ templ.SafeURL(b.URL) }>{ b.Title }</a>
        for _, tag := range b.Tags {
            <span data-bc="badge"
                  hx-get={ fmt.Sprintf("/bookmarks/filter?tag=%s", tag) }
                  hx-target="#bookmark-list"
                  hx-swap="innerHTML">
                { tag }
            </span>
        }
    </li>
}
```

### templ + HTMX Integration

HTMX attributes work naturally in templ files since they're just HTML attributes. The `hx-on:*` event attributes are also supported - templ treats them as script attributes with proper escaping:

```templ
<button hx-on:click={ templ.JSFuncCall("showMessage", "Hello from Go") }>Click me</button>
```

### The Build Workflow

```
1. Write .templ files
2. Run `templ generate` (compiles to .go files)
3. Go build picks up the generated .go files
```

`templ generate --watch` watches for changes and regenerates automatically.

Official Chi integration example: https://github.com/a-h/templ/tree/main/examples/integration-chi

## Asset Management

### Project Structure

```
project/
  static/
    css/        # Tailwind output
    js/         # htmx.min.js, stimulus controllers
    images/
    fonts/
  templates/
  main.go
```

### Serving Static Files

**Development** - serve from disk (no rebuild on CSS/JS change):

```go
r.Handle("/static/*", http.StripPrefix("/static/",
    http.FileServer(http.Dir("static"))))
```

**Production** - embed in binary:

```go
//go:embed static/*
var staticFiles embed.FS

r.Handle("/static/*", http.StripPrefix("/static/",
    http.FileServerFS(staticFiles)))
```

### CSS Build

Use the Tailwind standalone CLI (no Node required):

```bash
./tailwindcss -i src/input.css -o static/css/app.css --minify
```

### JS Loading

Stimulus controllers can be loaded as ES modules natively:

```html
<script type="module">
  import { Application } from "https://cdn.jsdelivr.net/npm/@hotwired/stimulus/dist/stimulus.js"
  import TaskFormController from "/static/js/controllers/task_form_controller.js"

  const app = Application.start()
  app.register("task-form", TaskFormController)
</script>
```

If bundling is needed, use esbuild (single binary, no Node required):

```bash
esbuild src/js/app.js --bundle --outfile=static/js/app.js --minify
```

### Cache Busting

```go
var version = "dev" // override at build: go build -ldflags "-X main.version=abc123"

func assetPath(path string) string {
    return fmt.Sprintf("/static/%s?v=%s", path, version)
}
```

```html
<link rel="stylesheet" href="{{assetPath "css/app.css"}}">
```

### Production

Put Nginx or a CDN in front:

```nginx
server {
    location /static/ {
        root /var/www/app;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    location / {
        proxy_pass http://localhost:3000;
    }
}
```

## Developer Experience (DX)

The DX is quite different from SvelteKit/Next where everything is one integrated system. You think server-first: the question isn't "what component do I build?" but "what URL returns the HTML I need?"

### Feature Development Flow

The full cycle for adding a new feature (e.g., "filter bookmarks by tag"):

```
1. Define route          (Go - 1 line)
2. Write handler         (Go - 3-5 lines)
3. Write template        (HTML - the actual UI)
4. Add hx-* attributes   (HTML - wire up interactivity)
5. Add Stimulus controller (JS - only if client behavior needed)
```

**Step 1 - Route:**

```go
r.Get("/bookmarks", listBookmarks)
r.Get("/bookmarks/filter", filterBookmarks) // HTMX partial
```

**Step 2 - Handler:**

```go
func filterBookmarks(w http.ResponseWriter, r *http.Request) {
    tags := r.URL.Query()["tag"]
    bookmarks, _ := db.GetBookmarksByTags(tags)
    tmpl.ExecuteTemplate(w, "bookmark-list", bookmarks)
}
```

No DTO, no serializer, no API versioning. Query the DB, render a template fragment.

**Step 3 & 4 - Template with HTMX attributes:**

```html
{{define "bookmark-list"}}
  {{range .}}
    <li data-bc="card">
      <a href="{{.URL}}">{{.Title}}</a>
      {{range .Tags}}
        <span data-bc="badge"
              hx-get="/bookmarks/filter?tag={{.}}"
              hx-target="#bookmark-list"
              hx-swap="innerHTML">{{.}}</span>
      {{end}}
    </li>
  {{end}}
{{end}}
```

Feature is functionally complete at this point - no JS written.

**Step 5 - Stimulus (only if client-side polish needed):**

For example, multi-tag filtering with active state on selected tags:

```javascript
// tag_filter_controller.js
export default class extends Controller {
  static targets = ["tag"]
  static values = { selected: Array }

  toggle(e) {
    const tag = e.currentTarget.dataset.tag
    const idx = this.selectedValue.indexOf(tag)
    if (idx === -1) this.selectedValue = [...this.selectedValue, tag]
    else this.selectedValue = this.selectedValue.filter(t => t !== tag)

    e.currentTarget.classList.toggle("bg-primary")
  }

  get filterUrl() {
    const params = this.selectedValue.map(t => `tag=${encodeURIComponent(t)}`).join("&")
    return `/bookmarks/filter?${params}`
  }
}
```

### Form Handling and Validation

In SvelteKit, libraries like svelte-superforms provide client-side validation, per-field error messages, form state management (dirty, submitting), and schema-based validation with Zod. The schema needs to be kept in sync between client and server.

In this stack, validation is entirely server-side. The server validates, returns errors as HTML, and HTMX swaps them in. No client-side validation library, no schema sync.

**On submit (all fields at once):**

```html
<form hx-post="/bookmarks"
      hx-status:422="target:#form-errors swap:innerHTML">
```

```go
errors := map[string]string{}
if title == "" {
    errors["title"] = "Title is required"
}
if len(errors) > 0 {
    w.WriteHeader(422)
    components.FormErrors(errors).Render(r.Context(), w)
    return
}
```

**Per-field validation (on blur):**

```html
<input name="url" type="url"
       hx-get="/validate/url"
       hx-trigger="blur"
       hx-target="next p"
       hx-swap="outerHTML" />
<p></p>
```

Each field validates independently as the user tabs through the form. The full form still validates on submit as a safety net.

**Feature comparison:**

| Feature | SvelteKit (superforms) | This Stack |
|---------|----------------------|------------|
| Per-field errors | Client-side, instant | Server round-trip on blur (~5ms) |
| Required indicators | Zod schema | HTML `required` attribute + CSS |
| Submitting state | `$submitting` store | HTMX `hx-indicator` or `hx-disabled-elt` |
| Schema validation | Zod (shared client/server) | Go struct tags (server only) |
| Client-side validation | Built-in | HTML5 attributes (`required`, `type`, `pattern`) |
| Validation logic location | Client + server (must sync) | Server only (single source of truth) |

The tradeoff: every validation round-trips to the server. For most forms this is imperceptible. For forms where instant client feedback matters (password strength, username availability), add a Stimulus controller or use HTML5 validation attributes.

The win: validation logic lives in one place. No keeping Zod schemas in sync between client and server. No hydration bugs where client validation passes but server rejects.

### DX Comparison with SvelteKit

| Step | SvelteKit | Go + HTMX |
|------|-----------|-----------|
| New feature starts with | `+page.svelte` + `+page.server.ts` | Go handler + HTML template |
| Data loading | `load()` function returns JSON | Handler renders HTML directly |
| UI | Svelte component with reactive markup | Go template with `hx-*` attributes |
| Interactivity | Runes, bindings, reactive blocks | HTMX attributes, Stimulus if needed |
| Styling | Scoped `<style>` in component | Basecoat `data-bc` + Tailwind classes |
| Testing the UI | Need browser/JSDOM | `curl` the endpoint, inspect HTML |
| Hot reload | Vite HMR (instant) | Air live reload (~200ms rebuild) |

### What You Lose vs SvelteKit (and Mitigations)

- **Hot reload** - use [Air](https://github.com/air-verse/air) for Go live reload. Watches `.go` and `.templ` files and restarts the server. Not as instant as Vite HMR but fast enough (~200ms).
- **Type safety in templates** - solved by using templ. Compile-time errors for wrong types, missing parameters, and broken component references. Full LSP support with autocomplete and go-to-definition.
- **Component dev/preview** - no Storybook equivalent out of the box. Build a `/dev` route that renders component examples.
- **Error overlay** - no browser error overlay. With templ, errors surface at compile time in your editor via LSP. Runtime errors show in Go server logs.

### What's Actually Better

- **Debugging** - `curl localhost:3000/bookmarks/filter?tag=go` and you see the exact HTML the user gets. No devtools, no network tab, no inspecting hydration. The response IS the UI.
- **Testing** - handler tests are plain Go HTTP tests. No browser needed, no JSDOM, no Playwright for basic functionality.
- **Mental overhead** - no thinking about client vs server boundaries, no "should this be a server action or client function?", no worrying about what gets serialized.

```go
func TestFilterBookmarks(t *testing.T) {
    req := httptest.NewRequest("GET", "/bookmarks/filter?tag=go", nil)
    w := httptest.NewRecorder()
    filterBookmarks(w, req)

    body := w.Body.String()
    if !strings.Contains(body, "Go Bookmarks") {
        t.Error("expected filtered results")
    }
}
```

### Recommended Dev Setup

```
terminal 1: templ generate --watch       # watches .templ files, generates .go
terminal 2: air                          # watches .go files, restarts server
terminal 3: tailwindcss -w -i ... -o ... # watches for CSS changes
editor:      any (VS Code, GoLand, Neovim) with templ LSP
browser:     just the app + devtools network tab
```

Three terminal processes. No `node_modules`. No build pipeline. The feedback loop is: edit `.templ` file -> templ generates Go code -> Air restarts (~200ms) -> refresh browser.

### The Honest DX Tradeoff

The DX is simpler but less polished. You trade Vite's instant HMR and Svelte's compiler feedback for a more transparent, debuggable system where every interaction is a visible HTTP request returning inspectable HTML. For developers who value understanding exactly what's happening over framework magic, this is a net positive. For those who rely heavily on IDE integration, type-safe templates, and instant feedback loops, there's a real adjustment period.

## PoC Application - Bookmark Manager

### App Name

**`stashd`** or **`markd`** - short, memorable, communicates the purpose.

### Why a Bookmark Manager

Goes beyond "hello world" (TodoMVC) while staying small. Exercises every part of the stack:

- Multiple related entities (bookmarks, folders, tags)
- Different interaction patterns (create, edit inline, delete, filter, search)
- Partial page updates (not just full page reloads)
- Client-side behavior that justifies Stimulus

### Stack Exercise Map

| Feature | HTMX | Basecoat | Stimulus | templ | Go/Chi |
|---------|------|----------|----------|-------|--------|
| Add bookmark | POST form, swap into list | Dialog, input, button | Open/close dialog, reset form | Form + list item components | CRUD route |
| Move between folders | GET with folder param, swap list | Sidebar, list | - | Folder layout + bookmark list components | Query + render |
| Live search | GET with debounce, swap results | Input | Keyboard shortcut (Ctrl+K) | Search results component | Search endpoint |
| Tag management | Include hidden inputs in requests | Badge components | Tag input add/remove | Tag badge component | Filter by tags |
| URL validation | - | Input states | Validate on paste | - | - |
| Folder tree | Nested GET requests | Tree/list components | Expand/collapse | Recursive folder component | Nested query |

## Project Structure

```
markd/
  cmd/
    server/
      main.go                  # HTTP server entry point (local dev)
      main_lambda.go           # Lambda entry point (build tag: lambda)
  internal/
    handler/
      handler.go               # Handler struct, constructor, shared helpers
      bookmarks.go             # Bookmark CRUD handlers
      folders.go               # Folder handlers
      search.go                # Search/filter handlers
      handler_test.go          # Unit tests for handlers
    model/
      bookmark.go              # Bookmark struct and types
      folder.go                # Folder struct and types
      tag.go                   # Tag struct and types
    store/
      store.go                 # Store interface
      sqlite.go                # SQLite implementation
      sqlite_test.go           # Store tests
    middleware/
      htmx.go                  # HX-Request detection, partial vs full page
      auth.go                  # Session/auth middleware
  components/
    layout.templ               # Base HTML layout (head, scripts, nav)
    bookmarks.templ            # Bookmark list, item, card components
    folders.templ              # Folder sidebar, tree components
    forms.templ                # Add/edit bookmark dialog, tag input
    search.templ               # Search results, filter bar
    shared.templ               # Shared partials (empty states, badges, errors)
  static/
    css/
      input.css                # Tailwind input (imports, base styles)
      app.css                  # Tailwind output (generated, gitignored)
    js/
      app.js                   # Stimulus app init + controller registration
      controllers/
        dialog_controller.js   # Open/close dialogs
        tag_input_controller.js # Tag add/remove behavior
        shortcuts_controller.js # Keyboard shortcuts
        chart_controller.js    # Chart rendering (if needed)
    vendor/
      htmx.min.js              # HTMX 4 (vendored)
      stimulus.min.js          # Stimulus (vendored)
  migrations/
    001_init.sql               # Create tables
  tests/
    integration/
      bookmarks_test.go        # Full CRUD flow tests with httptest.NewServer
      auth_test.go             # Auth middleware integration tests
    e2e/
      bookmarks.spec.js        # Playwright tests for Stimulus behavior
      playwright.config.js
  server-driven-ui-stack.md    # This document
  go.mod
  go.sum
  Makefile                     # Build, test, dev commands
  tailwind.config.js           # Tailwind + Basecoat config
  .air.toml                    # Air live reload config
```

### Key Decisions

- **`cmd/` and `internal/`** - standard Go project layout. `cmd/` has entry points, `internal/` has non-exportable packages.
- **`components/`** - templ files at the top level (not inside `internal/`) because `templ generate` works best with a dedicated component package. These are the UI layer.
- **`static/vendor/`** - HTMX and Stimulus vendored as JS files rather than pulled from CDN. Keeps the app self-contained and avoids external dependencies at runtime.
- **`internal/store/`** - interface-based so tests use SQLite in-memory and production can use SQLite file or swap to Postgres later.
- **`tests/integration/`** - separate from unit tests in `internal/` to keep the fast unit tests distinct from the slower (but still fast) integration tests.
- **`tests/e2e/`** - Playwright tests live outside Go. Only for the 5% that needs a real browser.

### Makefile Targets

```makefile
.PHONY: dev build test test-e2e generate css

generate:                          ## Generate templ Go code
	templ generate

css:                               ## Build Tailwind CSS
	tailwindcss -i static/css/input.css -o static/css/app.css --minify

build: generate css                ## Build the binary
	go build -o bin/markd ./cmd/server

dev:                               ## Run dev mode (3 watchers)
	@echo "Run these in separate terminals:"
	@echo "  templ generate --watch"
	@echo "  air"
	@echo "  tailwindcss -i static/css/input.css -o static/css/app.css --watch"

test:                              ## Run Go tests (unit + integration)
	go test ./...

test-e2e:                          ## Run Playwright E2E tests
	npx playwright test

lint:                              ## Run Go linter
	golangci-lint run
```

## Best Practices and Version Notes

**IMPORTANT**: When implementing, always use the latest stable versions and APIs of each library:

- **HTMX 4.x** - use the new event naming (`htmx:phase:action`), explicit inheritance (`:inherited`), `<hx-partial>` for multi-target updates, `hx-status` for error handling, morph swaps. Do NOT use deprecated HTMX 2.x patterns. Reference: https://four.htmx.org/docs
- **templ** - use the latest version. Write `.templ` files, run `templ generate`, use the LSP for IDE support. Prefer templ over `html/template` for type safety and composability. Reference: https://templ.guide/
- **Stimulus** - use the latest `@hotwired/stimulus` package. Follow the controller/target/value/action conventions. Reference: https://stimulus.hotwired.dev
- **Basecoat** - use the latest component APIs and `data-bc` attribute patterns. Reference: https://basecoatjs.com (verify current URL)
- **Go + Chi** - use the latest `github.com/go-chi/chi/v5`. Follow Go idioms, use `context` properly, handle errors explicitly. Reference: https://go-chi.io
- **Tailwind CSS** - use the latest v4.x with the standalone CLI where possible to avoid Node dependency. Reference: https://tailwindcss.com

### General Best Practices

- Server returns HTML fragments for HTMX requests, full pages for direct navigation
- Use `HX-Request` header to distinguish partial vs full page requests
- Use templ components as composable functions - keep them small, pass data via parameters
- templ components should be pure functions - don't rely on data not passed through parameters
- Keep Stimulus controllers small and focused - one behavior per controller
- Use Stimulus values (not data attributes) for passing server data to controllers
- Use HTMX events to bridge HTMX and Stimulus (e.g., `htmx:after:request->controller#method`)
- Prefer morph swaps (`innerMorph`/`outerMorph`) when preserving DOM state matters
- Use HTTP caching headers to reduce unnecessary server round-trips
- Embed static assets in the Go binary for production, serve from disk for development

## Docker Setup

Two-container setup: Nginx serves static assets with caching and compression, proxies HTML requests to the Go app.

### Project Structure Additions

```
markd/
  docker/
    nginx/
      nginx.conf             # Nginx config
      Dockerfile             # Nginx image
  Dockerfile                 # Go app image (multi-stage)
  docker-compose.yml         # Local production-like setup
```

### Go App Dockerfile

Multi-stage build - compiles Go + templ + Tailwind, produces a minimal image:

```dockerfile
# --- Build stage ---
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache curl

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Install Tailwind standalone CLI
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
    && chmod +x tailwindcss-linux-x64 \
    && mv tailwindcss-linux-x64 /usr/local/bin/tailwindcss

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate templ code
RUN templ generate

# Build Tailwind CSS
RUN tailwindcss -i static/css/input.css -o static/css/app.css --minify

# Build Go binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/markd ./cmd/server

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/markd /bin/markd
COPY --from=builder /app/migrations /migrations

EXPOSE 3000
CMD ["/bin/markd"]
```

No static assets in the runtime image - Nginx serves those.

### Nginx Dockerfile

```dockerfile
FROM nginx:alpine

COPY docker/nginx/nginx.conf /etc/nginx/nginx.conf
COPY static/ /usr/share/nginx/html/static/
```

### Nginx Config

```nginx
worker_processes auto;

events {
    worker_connections 1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile      on;

    # Gzip compression
    gzip on;
    gzip_types text/html text/css application/javascript application/json image/svg+xml;
    gzip_min_length 256;

    server {
        listen 80;

        # Static assets - served directly with long cache
        location /static/ {
            root /usr/share/nginx/html;
            expires 1y;
            add_header Cache-Control "public, immutable";
            access_log off;
        }

        # Favicon
        location = /favicon.ico {
            root /usr/share/nginx/html/static;
            expires 1y;
            access_log off;
        }

        # Everything else proxied to Go app
        location / {
            proxy_pass http://app:3000;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # Important for HTMX - don't buffer responses
            proxy_buffering off;
        }
    }
}
```

### docker-compose.yml

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      - DB_PATH=/data/markd.db
      - PORT=3000
    volumes:
      - db-data:/data
    expose:
      - "3000"

  nginx:
    build:
      context: .
      dockerfile: docker/nginx/Dockerfile
    ports:
      - "8080:80"
    depends_on:
      - app

volumes:
  db-data:
```

### Usage

```bash
# Build and run production-like setup
docker compose up --build

# Access at http://localhost:8080
```

### What This Gives You

- **Static assets never hit Go** - Nginx serves CSS, JS, images, fonts directly from disk with 1-year cache headers
- **Gzip compression** - Nginx compresses HTML, CSS, JS responses automatically
- **Single `docker compose up`** - reproducible production-like environment for testing
- **Same images deploy anywhere** - ECS/Fargate, EC2, any VPS, Docker Swarm
- **Go app stays simple** - no static file serving, no compression middleware, no cache header logic in production

### For ECS/Fargate Deployment

The same two containers run as a Fargate task definition. Nginx is the sidecar:

```
ALB (Application Load Balancer)
  -> Fargate Task
       -> Nginx container (port 80) -> Go app container (port 3000)
```

Or put CloudFront in front of the ALB for CDN-level caching of static assets.

## Deployment to AWS

### Lambda (Simplest Path)

Go on Lambda is straightforward. Go compiles to a single binary, Lambda runs it via the `provided.al2023` runtime. Cold starts are fast - typically 50-100ms, far better than Node or Java.

For an HTMX app, put API Gateway (HTTP API) in front of Lambda. Every request hits Lambda, which runs the Chi router, renders the templ component, and returns HTML.

The key library is `aws-lambda-go-api-proxy` - it adapts API Gateway events into standard `http.Request`/`http.ResponseWriter` so the Chi router works unchanged:

```go
package main

import (
    "context"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/awslabs/aws-lambda-go-api-proxy/chi"
    "github.com/go-chi/chi/v5"
)

var chiLambda *chiadapter.ChiLambda

func init() {
    r := chi.NewRouter()
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        components.Index().Render(r.Context(), w)
    })
    r.Get("/bookmarks", func(w http.ResponseWriter, r *http.Request) {
        bookmarks, _ := db.GetAll()
        components.BookmarkList(bookmarks).Render(r.Context(), w)
    })
    chiLambda = chiadapter.New(r)
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    return chiLambda.ProxyWithContext(ctx, req)
}

func main() {
    lambda.Start(handler)
}
```

Same routes, same handlers, same templ components. Keep a `main_local.go` for local dev with `http.ListenAndServe` and `main_lambda.go` with `lambda.Start`.

### Static Assets Architecture

Lambda returns HTML, but static assets should not go through it:

```
CloudFront (CDN)
  /static/*  -> S3 bucket (CSS, JS, images, fonts)
  /*         -> API Gateway -> Lambda (HTML)
```

Alternatively, use Lambda function URLs instead of API Gateway for simpler setup.

### Database Options for Lambda

| Option | Fit | Notes |
|--------|-----|-------|
| DynamoDB | Best | No connection pooling issues, pay-per-request |
| RDS/Aurora + RDS Proxy | Good | RDS Proxy manages connection pooling for Lambda |
| SQLite on EFS | OK | Low-concurrency only, ~5-10ms EFS latency |
| Turso/LibSQL | Good | SQLite-compatible, HTTP-based, no pooling needed |

### Lambda Strengths

- **Cold starts** - Go is the fastest Lambda runtime (50-100ms)
- **Cost** - for low-to-medium traffic, cheaper than running EC2/ECS 24/7
- **No server management** - no patching, no scaling config
- **Same code** - Chi router, templ components, Stimulus controllers are identical between local dev and Lambda

### Lambda Limitations

- **WebSockets** - HTMX SSE/WS extensions need API Gateway WebSocket APIs, which adds complexity
- **Response size** - capped at 6MB (usually fine for HTML)
- **Latency floor** - warm Lambda adds ~5-10ms overhead vs a long-running server; adds up in an HTMX app where every click is a round-trip

### Alternative: Fargate/ECS

If Lambda's constraints feel limiting, a single Fargate task running the Go binary is the next simplest option:

- Long-running process (WebSockets work natively)
- No cold starts
- Still no server management (no EC2 instances)
- Slightly higher base cost but predictable

For a PoC or low-traffic app, Lambda is the easiest path. For production with real-time features or consistent traffic, Fargate is the better fit.

### Reference

templ has an official AWS Lambda deployment example with CDK infrastructure code: https://github.com/a-h/templ/tree/main/examples/counter

## LLM-Friendliness

This stack is notably well-suited for LLM-assisted development compared to JS framework stacks.

### Why the Hyperstack Wins

- **Smaller, self-contained units of work** - a feature is a Go handler (5-10 lines), a templ component (10-20 lines), and maybe a Stimulus controller (15-30 lines). LLMs are much better at generating correct code when the scope is narrow. A SvelteKit page mixes server load functions, reactive state, template logic, scoped styles, and client-side effects in one file.
- **Less framework magic to get wrong** - HTMX is HTML attributes, templ is Go with HTML, Stimulus is vanilla JS with a thin convention layer. Very little implicit behavior to hallucinate. JS frameworks have compilers, reactivity transformations, hydration, and implicit file-based conventions that LLMs frequently get wrong.
- **Stable, smaller APIs** - HTMX has ~15 attributes, Stimulus has controllers/targets/values/actions, templ is a subset of Go. JS frameworks evolve rapidly (Svelte 4 to 5 changed the entire reactivity model) and LLM training data often lags behind, producing outdated patterns.
- **Standard HTTP semantics** - the entire interaction model is HTTP requests returning HTML. Every LLM understands HTTP deeply. Client-side state management and reactivity are where LLMs make the most mistakes.
- **Easier to verify** - `curl` an endpoint and inspect the HTML. No dev server, no browser, no inspecting hydration output.
- **Less context per feature** - a full feature is ~50-60 lines total. A SvelteKit equivalent can be 150+ lines across multiple files with implicit dependencies. Less context needed = more features per conversation = fewer mistakes from context window pressure.

### Where JS Frameworks Have an Edge

- **Training data volume** - vastly more React/Svelte/Next code in training data. For common patterns, LLMs can generate SvelteKit code almost from muscle memory.
- **Component ecosystem** - LLMs know exactly which Svelte/React library to reach for and how to wire it up. With Basecoat + Stimulus, more guidance may be needed.
- **Complex client-side logic** - for the 5% of features needing rich interactivity, LLMs produce better Svelte code than Stimulus controller code due to more training examples.

### Practical Consideration

HTMX 4 is new. LLMs may not have it in training data and may default to HTMX 2.x patterns or not know about it at all. However, the surface area is so small that this is easily solved by indexing the docs into a knowledge base:

- **HTMX 4** - ~15 attributes, ~20 events, a handful of config options. One page of docs covers most of it.
- **templ** - a subset of Go with HTML. The syntax guide is a single page.
- **Stimulus** - controllers, targets, values, actions. The handbook is 7 short pages.
- **Basecoat** - component catalog with attribute patterns.

The complete documentation for this entire stack is a fraction of what SvelteKit alone requires. The "training data gap" is a non-issue in practice - you close it with a few indexed pages.

## Testing

This stack is dramatically easier to test than any JS framework stack because every feature is just an HTTP endpoint returning HTML. The entire test suite runs with `go test ./...` - no browser binaries in CI, no Playwright setup, no flaky timeout-based assertions.

### 1. Unit Testing - templ Components

Components are pure functions. Pass in data, render to a string, assert on HTML output:

```go
func TestBookmarkItem(t *testing.T) {
    b := Bookmark{Title: "Go Blog", URL: "https://go.dev", Tags: []string{"go", "blog"}}
    var buf bytes.Buffer
    components.BookmarkItem(b).Render(context.Background(), &buf)

    html := buf.String()
    assert.Contains(t, html, "https://go.dev")
    assert.Contains(t, html, "Go Blog")
    assert.Contains(t, html, `hx-get="/bookmarks/filter?tag=go"`)
    assert.Contains(t, html, `data-bc="badge"`)
}

func TestBookmarkListEmpty(t *testing.T) {
    var buf bytes.Buffer
    components.BookmarkList([]Bookmark{}).Render(context.Background(), &buf)

    html := buf.String()
    assert.Contains(t, html, "No bookmarks yet")
    assert.NotContains(t, html, `data-bc="card"`)
}
```

No mounting, no virtual DOM, no component lifecycle. Runs in microseconds.

### 2. Unit Testing - Handlers

Test individual handlers with `httptest.NewRecorder`:

```go
func TestFilterBookmarks(t *testing.T) {
    db := setupTestDB()
    db.CreateBookmark(Bookmark{Title: "Learn Go", Tags: []string{"go"}})
    db.CreateBookmark(Bookmark{Title: "Learn Rust", Tags: []string{"rust"}})
    h := handler.New(db)

    req := httptest.NewRequest("GET", "/bookmarks/filter?tag=go", nil)
    w := httptest.NewRecorder()
    h.FilterBookmarks(w, req)

    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "Learn Go")
    assert.NotContains(t, w.Body.String(), "Learn Rust")
}
```

### 3. Unit Testing - HTMX Partial vs Full Page

Verify the server correctly distinguishes HTMX requests from direct navigation:

```go
func TestFullPageVsPartial(t *testing.T) {
    db := setupTestDB()
    h := handler.New(db)

    // Direct navigation - returns full page with <html>, <head>, etc.
    req := httptest.NewRequest("GET", "/bookmarks", nil)
    w := httptest.NewRecorder()
    h.ListBookmarks(w, req)
    assert.Contains(t, w.Body.String(), "<html")
    assert.Contains(t, w.Body.String(), "htmx.min.js")

    // HTMX request - returns fragment only
    req = httptest.NewRequest("GET", "/bookmarks", nil)
    req.Header.Set("HX-Request", "true")
    w = httptest.NewRecorder()
    h.ListBookmarks(w, req)
    assert.NotContains(t, w.Body.String(), "<html")
    assert.Contains(t, w.Body.String(), `id="bookmark-list"`)
}
```

### 4. Unit Testing - Form Submission and Validation

```go
func TestCreateBookmarkValidation(t *testing.T) {
    db := setupTestDB()
    h := handler.New(db)

    // Missing required field
    form := url.Values{"title": {""}, "url": {"https://go.dev"}}
    req := httptest.NewRequest("POST", "/bookmarks", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("HX-Request", "true")
    w := httptest.NewRecorder()
    h.CreateBookmark(w, req)

    assert.Equal(t, 422, w.Code)
    assert.Contains(t, w.Body.String(), "Title is required")
}

func TestCreateBookmarkSuccess(t *testing.T) {
    db := setupTestDB()
    h := handler.New(db)

    form := url.Values{"title": {"Go Blog"}, "url": {"https://go.dev"}, "tag": {"go"}}
    req := httptest.NewRequest("POST", "/bookmarks", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("HX-Request", "true")
    w := httptest.NewRecorder()
    h.CreateBookmark(w, req)

    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "Go Blog")
    assert.Contains(t, w.Body.String(), `hx-delete=`) // has delete button
}
```

### 5. Unit Testing - HTMX Response Headers

Verify the server sends correct HTMX response headers for triggers, redirects, etc.:

```go
func TestDeleteBookmarkTriggersCountUpdate(t *testing.T) {
    db := setupTestDB()
    db.CreateBookmark(Bookmark{ID: "42", Title: "Test"})
    h := handler.New(db)

    req := httptest.NewRequest("DELETE", "/bookmarks/42", nil)
    req.Header.Set("HX-Request", "true")
    w := httptest.NewRecorder()

    rctx := chi.NewRouteContext()
    rctx.URLParams.Add("id", "42")
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

    h.DeleteBookmark(w, req)

    assert.Equal(t, 200, w.Code)
    // Verify HX-Trigger header tells HTMX to update the count badge
    assert.Equal(t, "bookmarkCountChanged", w.Header().Get("HX-Trigger"))
}
```

### 6. Integration Testing - Full Request Flow

Spin up the actual Chi router in-process and make real HTTP requests:

```go
func setupTestServer() *httptest.Server {
    r := chi.NewRouter()
    db := setupTestDB() // SQLite in-memory
    h := handler.New(db)

    r.Get("/bookmarks", h.ListBookmarks)
    r.Post("/bookmarks", h.CreateBookmark)
    r.Get("/bookmarks/filter", h.FilterBookmarks)
    r.Delete("/bookmarks/{id}", h.DeleteBookmark)

    return httptest.NewServer(r)
}

func TestBookmarkCRUDFlow(t *testing.T) {
    ts := setupTestServer()
    defer ts.Close()
    client := ts.Client()

    // 1. Create a bookmark
    form := url.Values{"title": {"Go Blog"}, "url": {"https://go.dev"}, "tag": {"go"}}
    resp, _ := client.PostForm(ts.URL+"/bookmarks", form)
    assert.Equal(t, 200, resp.StatusCode)
    body, _ := io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "Go Blog")

    // 2. Verify it appears in the list
    resp, _ = client.Get(ts.URL + "/bookmarks")
    body, _ = io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "Go Blog")

    // 3. Filter by tag - should appear
    resp, _ = client.Get(ts.URL + "/bookmarks/filter?tag=go")
    body, _ = io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "Go Blog")

    // 4. Filter by wrong tag - should not appear
    resp, _ = client.Get(ts.URL + "/bookmarks/filter?tag=rust")
    body, _ = io.ReadAll(resp.Body)
    assert.NotContains(t, string(body), "Go Blog")
}
```

No Docker, no external process. The test starts a real HTTP server in-process, tests the full stack (router + middleware + handlers + templ + DB), and runs in milliseconds.

### 7. Integration Testing - Middleware

Test auth, CORS, rate limiting, etc. with the real middleware chain:

```go
func TestAuthMiddleware(t *testing.T) {
    ts := setupTestServer() // includes auth middleware
    defer ts.Close()
    client := ts.Client()

    // Unauthenticated - should redirect
    resp, _ := client.Get(ts.URL + "/bookmarks")
    assert.Equal(t, 302, resp.StatusCode)
    assert.Contains(t, resp.Header.Get("Location"), "/login")

    // Authenticated - should return page
    req, _ := http.NewRequest("GET", ts.URL+"/bookmarks", nil)
    req.AddCookie(&http.Cookie{Name: "session", Value: validSessionToken})
    resp, _ = client.Do(req)
    assert.Equal(t, 200, resp.StatusCode)
}
```

### 8. E2E Testing - Stimulus Behavior (Playwright, Only When Needed)

The 5% that needs a real browser - Stimulus controllers and actual DOM interactions:

```javascript
// tests/e2e/bookmarks.spec.js
import { test, expect } from '@playwright/test';

test('tag input adds and removes tags', async ({ page }) => {
    await page.goto('/bookmarks/new');

    // Type a tag and press Enter
    await page.fill('[data-tag-input-target="input"]', 'golang');
    await page.press('[data-tag-input-target="input"]', 'Enter');

    // Badge should appear
    await expect(page.locator('[data-bc="badge"]')).toContainText('golang');

    // Hidden input should exist for form submission
    await expect(page.locator('input[name="tag"][value="golang"]')).toBeAttached();

    // Click badge to remove
    await page.click('[data-bc="badge"]:has-text("golang")');
    await expect(page.locator('[data-bc="badge"]')).toHaveCount(0);
});

test('keyboard shortcut opens search', async ({ page }) => {
    await page.goto('/bookmarks');
    await page.keyboard.press('Control+k');
    await expect(page.locator('[data-shortcuts-target="searchInput"]')).toBeFocused();
});
```

This is the only layer that requires a browser. Reserve it for Stimulus interactions and critical user flows.

### Testing Pyramid for This Stack

```
        /  E2E  \          ~5% - Playwright for Stimulus/browser behavior
       /----------\
      / Integration \      ~25% - httptest.NewServer, full request flows
     /----------------\
    /    Unit Tests     \  ~70% - templ components + handlers + validation
   /--------------------\
```

All Go-based tests run with `go test ./...`. Fast, deterministic, no flaky browser timeouts. CI needs only Go installed for 95% of the test suite.

### Comparison with SvelteKit Testing

| Concern | Hyperstack | SvelteKit |
|---------|-----------|-----------|
| Unit test a component | Render to string, assert HTML | Need `@testing-library/svelte` + JSDOM |
| Test data loading | Plain HTTP test | Mock `load()` function or use test server |
| Test form submission | POST to handler, check response | Test form actions + progressive enhancement |
| Test validation | Check 422 response HTML | Mock server or test form action |
| Test interactivity | Verify HTMX attributes in HTML | Need browser/Playwright for reactive behavior |
| Test response headers | Check `w.Header()` | Need server-side test setup |
| Test auth flow | HTTP requests with cookies | Need to mock session + test hooks |
| Test full flow | HTTP requests with `HX-Request` header | Playwright end-to-end |
| Test speed | Milliseconds (no browser) | Seconds (JSDOM or browser startup) |
| CI complexity | `go test ./...` | Node + browser binaries + test runners |

## Summary

Most web applications are forms, tables, navigation, and filters - all well within what HTMX + Stimulus handles cleanly. The server-driven approach isn't a regression from SPAs - it's a mature alternative that trades client-side complexity for server-side simplicity.

This stack is best suited for: CRUD apps, dashboards, admin panels, content sites, internal tools, and multi-page apps where most logic lives on the server.

For the rare genuinely interactive widget (charts, drag-and-drop, collaborative editing), use Stimulus controllers as islands - JS only where it is needed, not everywhere by default.

The tools have caught up. HTMX 4's morphing, multi-target updates, and optimistic UI close the UX gaps that previously made SPAs the only viable choice. templ brings type safety and LSP support to server-side templates. Basecoat provides a component library without requiring React. The question is no longer "can you build real apps this way?" but "is this the right fit for your app?"
