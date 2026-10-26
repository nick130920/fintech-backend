# Upgrade Go toolchain to 1.26.10

## Goal

Restore CI after `govulncheck` reported `GO-2026-6809` in `crypto/x509@go1.26.9`, fixed in Go 1.26.10.

## Scope

- Update the declared Go toolchain in `go.mod`.
- Update the Docker builder image.
- Update the GitHub Actions Go version.
- Validate tests, build, vet, and vulnerability scanning with Go 1.26.10.
- Publish the resulting work-unit commit to PR #44 after native review.

## Non-goals

- No application behavior changes.
- No dependency upgrades beyond the Go patch toolchain.
- No merge or deployment.

## Tasks

- [x] Update Go 1.26.9 references to 1.26.10.
- [x] Verify toolchain, tests, build, vet, formatting, and vulnerability scan.
- [ ] Commit, review, and publish the work unit to `fix/sms-processing-failures`.

## Evidence

- Changed declarations:
  - `.github/workflows/ci.yml`: `go-version: '1.26.9'` → `go-version: '1.26.10'`.
  - `Dockerfile`: `FROM golang:1.26.9-alpine AS builder` → `FROM golang:1.26.10-alpine AS builder`.
  - `go.mod`: `toolchain go1.26.9` → `toolchain go1.26.10`.
- Preserved `go 1.26.0` and all module versions.
- GREEN verification with `GOTOOLCHAIN=auto`:
  - `go version`: `go1.26.10 linux/amd64`.
  - `go test ./...`: passed (13 tested packages; 10 packages without tests).
  - `go build ./...`: passed.
  - `go vet ./...`: passed.
  - `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`: passed; code affected by 0 vulnerabilities.
  - `git diff --check`: passed.
  - `docker build -t fintech-backend:go-1.26.10-check .`: passed; server compiled with the 1.26.10 Alpine builder.
- RED: GitHub Actions runs for PR #44 and PR #45 both failed in `Check vulnerabilities` with `GO-2026-6809`; affected `crypto/x509@go1.26.9`, fixed in `crypto/x509@go1.26.10`.
