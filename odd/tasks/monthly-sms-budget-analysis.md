# Monthly SMS budget analysis

## Outcome

Allow the mobile app to submit every locally included SMS from the complete previous calendar month without silent truncation, while preserving strict OpenRouter ZDR routing and keeping raw SMS payloads ephemeral.

## Decisions

- The mobile app selects the previous calendar month using device-local calendar boundaries.
- The mobile app displays full included/excluded SMS only on-device; excluded SMS never reach this API.
- The backend accepts at most 500 included SMS per job and rejects larger requests explicitly.
- Processing retains 25-message chunks and existing retry, partial-success, sanitization, context, status, and error semantics.
- Raw submitted messages are passed to the in-process worker but are not persisted in `budget_suggestion_jobs.messages_json`.
- No provider inference or production deployment is part of this work unit.

## Tasks

- [x] MB-1 — Confirm current DTO, handler, job persistence, 100-message cap, chunking, timeout, and strict ZDR behavior against `origin/main=b9a5d204463a65896e85311d7a7f99516a8dc796`.
  - Evidence: request DTO has body/date only; server truncates before filtering at 100 and processes at most 4×25; jobs persist raw `MessagesJSON`; strict budget requests use `deepseek/deepseek-v4.1-flash` with `provider.zdr=true`.
- [x] MB-2 — Add failing tests for >100 complete processing, 500-message boundary, 501 rejection before persistence, and ephemeral job payload.
  - Observed behavioral RED with Go 1.26.10 before production edits: 101/126/500 processed only 100 messages in four chunks; created job retained raw SMS; 501 reached repository Create and returned a repository error instead of a safe 400.
  - Tests inspect actual numbered synthetic input for exact 25-message chunking and once-only delivery, verify independent worker copy after caller mutation, and cover status ownership, empty input, and zero-expense completion.
- [x] MB-3 — Implement bounded complete chunking and non-persistence while preserving strict routing and failure semantics.
  - Shared named maximum of 500 validated before job creation/worker launch and before direct processing. Over-limit input returns a fresh safe invalid-request AppError (400); no truncation. Full filtered input traversed in 25-message chunks.
  - Job creation stores empty `MessagesJSON` in the existing NOT NULL text column; raw SMS remain only in the independent in-process worker copy. No migration needed.
  - Strict provider implementation, retry schedule, sanitization, partial success, failure/context/error ownership semantics, and 12-minute deadline preserved.
- [x] MB-4 — Run focused tests, race checks, full tests, vet, formatting, diff checks, and independent verification.
  - Independent verifier passed the full suite, focused race suite, `go vet`, formatting check, and `git diff --check`; no blocker found.
  - Delivery inventory explicitly includes `internal/usecase/sms_suggestion_job_capacity_test.go` and this task document.
- [ ] MB-5 — Commit the verified backend work unit with tests and this document.
  - Blocked pending explicit user authorization to create the work-unit commit.

## Verification evidence

- RED attempt: `GOPROXY=off GOTOOLCHAIN=local go test ./internal/usecase -run 'Test(ProcessSMSBatchForSuggestions|SMSBatchSuggestionJob|StartSMSBatchSuggestionJob|GetSMSBatchSuggestionJobStatus)' -count=1` — exit 1 before compilation: `go: go.mod requires go >= 1.26.0 (running go 1.22.2; GOTOOLCHAIN=local)`. No behavioral RED observed.
- Race attempt: `GOPROXY=off GOTOOLCHAIN=local go test -race ./internal/usecase -run 'Test(ProcessSMSBatchForSuggestions|SMSBatchSuggestionJob|StartSMSBatchSuggestionJob|GetSMSBatchSuggestionJobStatus)' -count=1` — same Go version gate, exit 1; no tests executed.
- Behavioral RED: `GOPROXY=off GOTOOLCHAIN=auto go test ./internal/usecase -run 'Test(ProcessSMSBatchForSuggestions|SMSBatchSuggestionJob|StartSMSBatchSuggestionJob|GetSMSBatchSuggestionJobStatus)' -count=1` — exit 1, tests compiled and exposed expected old behavior: 4 calls instead of 5/6/20 for 101/126/500 messages, persisted raw SMS, and Create called once for 501. Existing regression cases passed.
- GREEN: same exact `GOTOOLCHAIN=auto` command after minimal production implementation — passed (`ok`, 0.338s).
- Race GREEN: `GOPROXY=off GOTOOLCHAIN=auto go test -race ./internal/usecase -run 'Test(ProcessSMSBatchForSuggestions|SMSBatchSuggestionJob|StartSMSBatchSuggestionJob|GetSMSBatchSuggestionJobStatus)' -count=1` — passed (`ok`, 5.181s). Includes existing partial-success/all-failed/context-sentinel/safe-error cases plus new capacity/privacy/ownership cases. Provider HTTP requests were intercepted locally using synthetic inputs and credentials; no inference requests left the process.
- Post-tidy verification: repeated the exact focused `GOTOOLCHAIN=auto` commands above; normal tests passed (`ok`, 0.812s) and race tests passed (`ok`, 6.862s).
- Capacity risk: keep the 12-minute deadline. At 500 messages / 25 per chunk = 20 sequential chunks, worst scheduled retry waits total 20 × (0+3+8+15) = 520 seconds, leaving 200 seconds for provider latency and overhead. This margin is not a completion guarantee; slow provider attempts can exhaust the deadline.
- Independent full suite: `GOTOOLCHAIN=auto go test ./...` — passed; `internal/usecase` and `internal/usecase/webapi` passed.
- Independent focused race: `GOTOOLCHAIN=auto go test -race ./internal/usecase -run 'Test(ProcessSMSBatchForSuggestions|SMSBatchSuggestionJob|StartSMSBatchSuggestionJob|GetSMSBatchSuggestionJobStatus)' -count=1` — passed; no race reported.
- Independent static/structural checks: `GOTOOLCHAIN=auto go vet ./...`, `gofmt -l` for all changed Go files, and `git diff --check` — passed with no output.
- No provider/network inference, dependency mutation, schema change, production access, database access, or commit performed.

## Commit evidence

Pending.
