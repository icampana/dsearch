# dsearch

**dsearch** is a fast, offline documentation search tool for your terminal. It brings the power of [DevDocs.io](https://devdocs.io) to your command line, allowing you to search and read documentation without leaving your terminal.

![License](https://img.shields.io/badge/license-MPL_2.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.25.7-blue.svg)

## Features

- ⚡ **Instant Fuzzy Search**: Find what you need quickly, even with typos.
- 📚 **Offline Access**: Download once, search anywhere. No internet required after installation.
- 🖥️ **Terminal-First Experience**: Read docs directly in your terminal with rich formatting.
- 📝 **Markdown Output**: Pipe content to your favorite markdown viewer or editor (e.g., `glow`, `mdcat`, `vim`).
- 🎯 **Scoped Search**: Search across all installed docs or filter to a specific framework (e.g., just `react`).
- 🔧 **XDG Compliant**: Follows standard Linux/Unix directory structures for config and data.

## Installation

### From Source

If you have Go installed (1.23+), you can install `dsearch` directly:

```bash
go install github.com/icampana/dsearch/cmd/dsearch@latest
```

Ensure your `$GOPATH/bin` is in your `$PATH`.

### Using eget

You can either use [eget](https://github.com/zyedidia/eget)

Easiest way to install without go (Linux/macOS):

```bash
eget icampana/dsearch --to=~/.local/bin
```

### Pre-built Binaries

Pre-built binaries for Linux, macOS, and Windows are available on the [Releases](https://github.com/icampana/dsearch/releases) page.

## Usage

### 1. Install Documentation

Before searching, you need to download documentation sets from DevDocs.

```bash
# Install specific docs
dsearch install go react

# Install with version/release
dsearch install react@18
dsearch install python~3.11
```

### 2. Search

```bash
# Search across all installed docs
dsearch useState

# Search within a specific doc
dsearch -d react useState

# detailed output (full content)
dsearch -d go http.Client --full
```

### 3. Output Formats

By default, `dsearch` outputs simplified text with terminal formatting. You can also output Markdown or JSON.

```bash
# Output as Markdown (great for piping to glow)
dsearch -f md useState | glow -

# Output results as JSON (for scripting)
dsearch --json useState
```

## Configuration

`dsearch` follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).

- **Data (Docs)**: `$XDG_DATA_HOME/dsearch` (default: `~/.local/share/dsearch`)
- **Cache**: `$XDG_CACHE_HOME/dsearch` (default: `~/.cache/dsearch`)
- **Config**: `$XDG_CONFIG_HOME/dsearch` (default: `~/.config/dsearch`)

## AI Agent Skill

This project includes a specialized skill for AI agents (like those using `vercel-labs/skills`). This allows agents to autonomously search and read documentation.

To install the skill:

```bash
npx skills add https://github.com/icampana/dsearch
```

This will expose the `dsearch` tool to your agent, allowing it to verify syntax, check signatures, and read docs without internet access.

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.

---

*Not affiliated with DevDocs.io, but huge thanks to them for their amazing API and documentation aggregation.*
