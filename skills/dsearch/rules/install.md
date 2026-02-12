---
name: dsearch-cli-installation
description: |
  Install the dsearch CLI tool automatically using eget or other methods.
---

# dsearch CLI Installation

## Quick Install (Recommended)

Using [eget](https://github.com/zyedidia/eget) - the fastest way to install pre-built binaries:

```bash
eget icampana/dsearch --to=~/.local/bin
```

**Note**: If `eget` is not installed, install it first:

```bash
brew install eget
```

## Alternative Installation Methods

### From Source (Go Required)

If you have Go 1.23+ installed:

```bash
go install github.com/icampana/dsearch/cmd/dsearch@latest
```

Ensure `$GOPATH/bin` is in your `$PATH`.

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/icampana/dsearch/releases):

```bash
# Example for macOS ARM64
curl -L https://github.com/icampana/dsearch/releases/latest/download/dsearch_Darwin_arm64.tar.gz | tar xz
mv dsearch ~/.local/bin/
```

## Verify Installation

Check if installed correctly:

```bash
dsearch --version
```

## Post-Installation Setup

After installation, install your first documentation set:

```bash
# Install common docs
dsearch install go react python

# Or install specific versions
dsearch install react@18
dsearch install python~3.11
```

## Troubleshooting

### Command not found

If `dsearch` command is not found after installation:

1. **Check if `~/.local/bin` is in PATH**:
   ```bash
   echo $PATH | grep -q "$HOME/.local/bin" || echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
   source ~/.zshrc
   ```

2. **Verify the binary exists**:
   ```bash
   ls -la ~/.local/bin/dsearch
   ```

3. **Try reinstalling with eget**:
   ```bash
   eget icampana/dsearch --to=~/.local/bin --upgrade
   ```

### Permission errors

If you get permission errors:

```bash
chmod +x ~/.local/bin/dsearch
```

### eget not found

If `eget` is not installed:

```bash
# macOS
brew install eget

# Linux (using eget's install script)
curl https://zyedidia.github.io/eget.sh | sh
mv eget ~/.local/bin/
```

## Automatic Installation for Agents

When an agent detects that `dsearch` is not available in PATH, it should:

1. **Check for eget**: Run `which eget`
2. **If eget exists**: Run `eget icampana/dsearch --to=~/.local/bin`
3. **If eget missing**: Install eget first with `brew install eget`, then install dsearch
4. **Verify**: Run `dsearch --version` to confirm installation
5. **Setup**: Prompt user to install initial documentation sets

This ensures a seamless installation experience without manual intervention.
