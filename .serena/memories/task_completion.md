# Task Completion Checklist

## Before Completing a Task

### 1. Code Quality
- [ ] Run `go fmt ./...` - ensure all code is formatted
- [ ] Run `go vet ./...` - check for common Go errors
- [ ] Run `golangci-lint run` - comprehensive linting
- [ ] Run `go test ./...` - all tests pass
- [ ] Run `go test -race ./...` - no race conditions

### 2. Build Verification
- [ ] `make build` succeeds without errors
- [ ] Binary runs successfully (`./bin/dsearch`)
- [ ] Test the specific feature/fix manually
- [ ] Verify no warnings during compilation

### 3. Error Handling
- [ ] All errors are checked and handled
- [ ] Errors are wrapped with context using `fmt.Errorf("...: %w", err)`
- [ ] No panic/recover in production code
- [ ] Resource cleanup with `defer` where needed

### 4. Documentation
- [ ] Exported functions have godoc comments
- [ ] Complex logic has inline comments
- [ ] Update README if user-facing changes
- [ ] Update CHANGELOG.md under `[Unreleased]`

### 5. Testing
- [ ] Unit tests for new functionality
- [ ] Table-driven tests for multiple scenarios
- [ ] Test both success and error paths
- [ ] Integration tests if applicable

### 6. Type Safety
- [ ] No use of `interface{}` unless absolutely necessary
- [ ] No type assertions without checking ok
- [ ] Prefer named types over anonymous structs
- [ ] Use context.Context for cancellation

## After Completing a Task

### 1. Git Workflow
```bash
# Stage changes
git add .

# Check status
git status

# Review diff
git diff --staged

# Commit with conventional commit message
git commit -m "feat(tui): add interactive search interface"
```

### 2. Commit Message Format

Follow Conventional Commits specification:
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Adding/updating tests
- `chore`: Build process, tooling

**Examples:**
```
feat(search): add fuzzy matching with score sorting

Implements fuzzy search using sahilm/fuzzy library.
Results are sorted by relevance score (0-1) and name.

Closes #123
```

```
fix(tui): handle window resize events

Fixed issue where TUI would crash on terminal resize.
Now properly updates model dimensions and component sizes.
```

### 3. Update Documentation

**README.md** - if user-facing:
```markdown
## Usage

Interactive mode:
```bash
dsearch
```
```

**CHANGELOG.md** - always update:
```markdown
## [Unreleased]

### Added
- Interactive TUI for browsing search results
- Fuzzy matching with relevance scoring

### Fixed
- Window resize handling in TUI
```

### 4. Clean Up
- [ ] Remove debug `fmt.Printf` statements
- [ ] Remove unused variables
- [ ] Remove commented-out code
- [ ] Clean up imports (`go mod tidy`)

### 5. Verification Steps

**For Bug Fixes:**
1. Reproduce the bug before the fix
2. Apply the fix
3. Verify the bug is resolved
4. Test edge cases

**For New Features:**
1. Write tests for the feature
2. Implement the feature
3. Verify all tests pass
4. Manual testing in production-like environment

**For Refactoring:**
1. Ensure tests still pass
2. Run performance benchmarks if applicable
3. Verify behavior is unchanged

## Common Gotchas

### Bubble Tea TUI Development
- [ ] Always handle `tea.Quit` command
- [ ] Handle `tea.WindowSizeMsg` for responsive UI
- [ ] Use `tea.Batch` for multiple commands
- [ ] Don't block in `Update()` method
- [ ] Use commands for async operations

### Concurrent Code
- [ ] Run `go test -race` to detect data races
- [ ] Always use context for cancellation
- [ ] Close channels properly
- [ ] Use WaitGroups for goroutine synchronization

### Error Handling
- [ ] Never ignore errors (`_, err := ...`)
- [ ] Wrap errors with context
- [ ] Check sentinel errors with `errors.Is()`
- [ ] Extract error types with `errors.As()`

### Database Operations
- [ ] Always close connections with `defer`
- [ ] Use prepared statements
- [ ] Handle `sql.ErrNoRows` explicitly
- [ ] Use read-only mode for queries when possible

## Pull Request Process

### Before Creating PR
1. Update CHANGELOG.md
2. Ensure all tests pass
3. Update documentation
4. Review diff for unwanted changes
5. Create a feature branch from `main`

### PR Description Template
```markdown
## Description
Brief description of changes.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Manual testing performed
- [ ] Integration tests pass

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] No new warnings generated
```

## Post-PR Checklist

### After Merge
- [ ] Delete feature branch
- [ ] Update local main branch
- [ ] Verify production deployment
- [ ] Close related issues

## Continuous Improvement

### Regular Tasks (Weekly/Monthly)
- [ ] Update dependencies (`go get -u ./... && go mod tidy`)
- [ ] Check for security vulnerabilities (`go list -json -m all | nancy sleuth`)
- [ ] Review and update documentation
- [ ] Refactor technical debt
- [ ] Performance benchmarking

### Code Review
- [ ] Review code for idiomatic Go patterns
- [ ] Check for unnecessary complexity
- [ ] Verify error handling completeness
- [ ] Ensure tests are comprehensive
- [ ] Validate performance implications
