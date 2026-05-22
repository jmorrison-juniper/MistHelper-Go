# Changelog

All notable changes to MistHelper-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions use UTC timestamp format: `YY.MM.DD.HH.MM`.

## [Unreleased]

### Added
- Initial project scaffolding (Go rewrite of MistHelper Python)
- CI/CD workflows: quality gates, container build, release, CodeQL
- Multi-stage Containerfile (~25MB image vs ~500MB Python)
- GitHub issue/PR templates adapted for Go tooling
- Dependabot configuration for Go modules and GitHub Actions
- SpecKit agent and prompt files for structured development
- Deploy files: Podman Quadlet, systemd service, .env.example
- Copilot Coding Agent setup steps for Go environment
