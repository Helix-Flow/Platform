# AGENTS.md

HelixFlow AI inference platform configuration repository with Specify development workflow.

## Development Commands

### Feature Management
- Create new feature: `.specify/scripts/bash/create-new-feature.sh <feature-name>`
- Check prerequisites: `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- Setup implementation plan: `.specify/scripts/bash/setup-plan.sh`
- Update agent context: `.specify/scripts/bash/update-agent-context.sh`

### Testing
- Single test: No framework configured - check plan.md for test commands
- All tests: Run after implementation based on tech stack in plan.md

### CI/CD and Quality Assurance
- Manual Execution: All CI/CD processes must be executed manually
- No GitHub Actions: No automated workflows configured
- No Git Hooks: No pre-commit, pre-push, or other automated Git hooks
- Manual Quality Gates: All testing, linting, and reviews performed manually
- Explicit Commands: Developers must run all checks before commits and deployments

### Git Operations
- Upstream: git@github.com:Helix-Flow/Platform.git
- Feature branches: Must use 3-digit prefix (e.g., 001-feature-name)
- Current branch detection: Uses SPECIFY_FEATURE env var or git branch

## Code Style Guidelines

### Bash/Shell Scripts (.sh)
- Use `#!/usr/bin/env bash` shebang
- Source `common.sh` for shared functions (get_repo_root, get_current_branch, etc.)
- Use absolute paths from `get_repo_root()`
- Handle both git and non-git repositories gracefully
- Follow existing function patterns: `check_*`, `get_*`, `find_*`

### File Organization
- Scripts: `.specify/scripts/bash/` with executable permissions
- Templates: `.specify/templates/` with `.md` extension
- Feature specs: `specs/XXX-feature-name/` directories
- Opencode commands: `.opencode/command/` with `.md` extension

### Error Handling
- Explicit error checking with meaningful messages
- Use `>&2` for error output
- Return appropriate exit codes
- Validate prerequisites before execution
- Never commit secrets or API keys

### Naming Conventions
- Feature branches: 3-digit prefix with hyphen (001-feature-name)
- Functions: snake_case with descriptive names
- Variables: UPPER_SNAKE_CASE for exports, lower_snake_case for locals
- Files: kebab-case for directories, descriptive names for scripts

### Project Structure
- Root: Platform configuration and upstream tracking
- `.specify/`: Development workflow automation
- `.opencode/`: AI agent command definitions
- `specs/`: Feature specifications and implementation plans
- `Upstreams/`: Remote repository configuration