---
name: linting-formatting
description: Run golangci-lint and gofumpt for code quality. Use when linting or formatting Go code.
---

# Linting & Formatting

Skill for running golangci-lint and gofumpt on the Go codebase.

## golangci-lint

### Run Linter
```bash
cd ./garden-app && golangci-lint run
```

### Configuration

Configuration is in `./garden-app/.golangci.yaml`:

**Enabled Linters:**
- `gofumpt` - Stricter gofmt
- `gosec` - Security issues
- `revive` - Fast linter (disabled for test files)
- `errcheck` - Unchecked errors
- `gosimple` - Simplification suggestions
- `staticcheck` - Static analysis
- `unused` - Unused code
- `typecheck` - Type checking
- `misspell` - Spelling mistakes
- `gocritic` - Opinionated checks
- `ineffassign` - Ineffective assignments
- `nakedret` - Naked returns
- And more...

**Excluded from errcheck:**
- `render.Render` errors
- `viper.BindPFlag` errors

## gofumpt

### Format Code
```bash
cd ./garden-app && gofumpt -w .
```

Or via Task:
```bash
task gofumpt
```

### Characteristics

- Stricter than `gofmt`
- Enforces vertical alignment of field lists
- Groups `const` and `var` declarations
- Removes empty lines before function endings

## CI Integration

Both linters run in GitHub Actions on every PR. PRs cannot be merged if linting fails.

## Fixing Issues

1. Run `task gofumpt` first (fixes formatting automatically)
2. Run `task lint` to see remaining issues
3. Fix issues manually or with `golangci-lint run --fix` (where supported)
