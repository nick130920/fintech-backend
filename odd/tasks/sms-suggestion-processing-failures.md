# SMS suggestion processing failures

## Goal

Prevent asynchronous SMS budget suggestion jobs from reporting a successful empty result when processing did not produce a trustworthy outcome.

## Confirmed production evidence

- The mobile client submitted 100 locally selected SMS messages.
- The async response contract correctly exposes `suggestions` at the response root.
- The reproduced production job encountered OpenRouter rate limits and a ZDR/privacy-policy routing rejection.
- `ProcessSMSBatchForSuggestions` currently skips failed AI chunks and can return an empty successful response.
- No SMS bodies, senders, amounts, codes, or credentials were retained during diagnosis.
- Independent verification of the uncommitted candidate found that OpenRouter batch errors embedded raw provider response bodies, which the fallback wrapper logged. Malformed batch model JSON was also logged verbatim. Aggregate usecase logging alone did not make the executed pipeline privacy-safe.

## Scope

- Distinguish unavailable categories, zero usable messages, total AI-processing failure, partial AI-processing success, and a genuine processed result with zero accepted expenses.
- Fail the job when no non-empty chunk is processed successfully.
- Preserve partial results when at least one chunk succeeds, while emitting privacy-safe aggregate observability.
- Log counts only: received, filtered, attempted chunks, successful chunks, failed chunks, and accepted expenses.
- Add deterministic unit tests for total failure and partial success.
- Sanitize OpenRouter batch HTTP errors, provider-error envelopes, and budget JSON parse failures before errors reach fallback logging. Preserve status/error classification and existing rate-limit retries.

## Non-goals

- Log or persist SMS bodies, senders, amounts, codes, prompts, or model responses.
- Change the mobile request payload.
- Tune the classifier or category taxonomy in this work unit.
- Change OpenRouter account privacy settings or provider credentials.

## Tasks

- [x] Add failing tests for total AI failure and partial AI success.
- [x] Implement explicit failure semantics and aggregate privacy-safe observability.
- [x] Run focused backend tests and static checks.
- [x] Reproduce the provider-log privacy leak with failing batch extraction/fallback-log tests.
- [x] Remove raw provider/model content from batch budget failure logs and returned errors; verify retries, successful parsing, and cancellation remain intact.

## Evidence

### RED (before implementation)

Command:

```sh
go test ./internal/usecase -run '^TestProcessSMSBatchForSuggestions_(TotalAIFailure|PartialSuccess|ProcessedWithoutAcceptedExpenses|CategoryFailures|NoFilteredMessages)$' -count=1
```

Observed exit 1:

- Total AI failure returned a non-nil empty response with a nil error.
- Both partial-success orderings retained the accepted expense, but failed the new aggregate-count log assertion because no summary was emitted.
- Category repository errors and genuinely empty categories returned empty success instead of errors.
- Legitimately empty processing and filtered-empty input failed only their missing aggregate-count log assertions.

### GREEN (after implementation)

The exact RED command above passed: `ok .../internal/usecase 0.076s`.

### Focused validation and alternate cases

```sh
go test ./internal/usecase -run '^(TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1
```

Passed: `ok .../internal/usecase 0.063s`.

```sh
go test -race ./internal/usecase -run '^(TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1
```

Passed: `ok .../internal/usecase 1.587s`, no race reports.

Coverage includes total failure, partial success in either chunk order, duplicate/rejected lines, successful processing with zero accepted expenses (empty lines or income-only), unavailable categories, filtered-empty messages without a category dependency, the 100-message/four-chunk cap, and the async job transition from processing to failed with no result JSON.

Tests intercept HTTP with deterministic synthetic responses and synthetic credentials; no live provider calls or runtime SMS data are used. The failed-job error does not expose the synthetic provider error body. This original assertion covered the returned job error, not provider/fallback logs; the privacy correction below covers those logs explicitly. Exact summary assertions permit only six aggregate counts; `filtered_messages` means retained messages after filtering the capped sample, while `received_messages` counts the entire input.

### Static checks

- `go vet ./internal/usecase`: exit 0, no findings.
- `gofmt -l internal/usecase/bank_notification_pattern.go internal/usecase/sms_suggestion_processing_test.go internal/usecase/sms_suggestion_job_test.go`: exit 0, no unformatted files.
- `git diff --check`: exit 0, no whitespace errors.

### Scope notes

- Zero retained messages return a successful empty response before categories are fetched; categories are required only when AI work is needed.
- Non-empty chunks increment attempted counts immediately before the AI call. Errors or nil results increment failures; any non-nil error-free response counts as success, independently of accepted expenses.
- The initial aggregate change left provider-layer logging untouched; the follow-up privacy correction below fixes the confirmed batch-provider leaks. Slug-stat infrastructure remains unchanged. The batch summary adds no bodies, senders, amounts, prompts, raw responses, credentials, user identifiers, or job identifiers.
- Classifiers, prompts, provider configuration, and DTO contracts are unchanged. No commit was created.

### Provider-log privacy correction — test-first evidence

RED command (before the provider correction):

```sh
go test ./internal/usecase/webapi -run '^TestOpenRouterBatchBudget' -count=1
```

Observed failure: synthetic private provider content appeared in returned HTTP/provider-envelope errors and fallback logs; malformed model output appeared in parse logs. The new classification assertions also rejected unsanitized provider errors and unclassified malformed envelopes.

GREEN: the same command passed after correction (`ok .../internal/usecase/webapi 0.012s`).

The regression tests capture both OpenRouter and the actual `AIServiceWithFallback` logger. Cases cover HTTP 403/503 bodies, HTTP 429 exhaustion, 200+provider-error envelopes, rate-limit/429 envelope exhaustion, malformed provider envelopes, malformed model JSON, and model JSON type errors. Alternate cases cover successful parsing after HTTP/envelope rate limits and cancellation during backoff. `testing/synctest` exercises all four retry attempts with virtual time; production delays and provider settings are unchanged.

Sanitized errors retain HTTP status or a fixed provider/parse classification (plus numeric provider code where available). No decoder error is wrapped for the budget model or provider envelope, because decoder diagnostics can contain model-controlled fields/values. The shared batch HTTP helper is sanitized for both batch consumers; transaction-model parsing and single-SMS extraction remain outside this correction's scope. Prompts, payloads, successful parsing, provider settings, and retry classification are unchanged.

Validation after correction:

- `go test ./internal/usecase/webapi -count=1`: passed (`0.013s`).
- `go test ./internal/usecase -run '^(TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1`: passed (`0.051s`).
- `go test -race ./internal/usecase/webapi ./internal/usecase -run '^(TestOpenRouterBatchBudget|TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1`: passed (provider `1.049s`, usecase `1.526s`), no race reports.
- `go vet ./internal/usecase/webapi ./internal/usecase`: exit 0, no findings.
- Formatting/diff checks: `gofmt -w internal/usecase/webapi/openrouter_service.go internal/usecase/webapi/openrouter_batch_test.go` completed; `gofmt -l internal/usecase/webapi/openrouter_service.go internal/usecase/webapi/openrouter_batch_test.go internal/usecase/bank_notification_pattern.go internal/usecase/sms_suggestion_processing_test.go internal/usecase/sms_suggestion_job_test.go` emitted no paths. `git diff --check` passed with no whitespace errors.

### Second privacy correction — complete budget primary/fallback pipeline

Independent follow-up checks confirmed remaining leaks: OpenRouter transport/body-read errors survived retry exhaustion, Gemini wrapped SDK errors and logged malformed model output plus decoder diagnostics, and the fallback wrapper logged/returned raw provider errors. The earlier tests had no configured fallback and did not exercise those I/O failures.

RED command, run after adding `budget_privacy_test.go` and before implementation:

```sh
go test ./internal/usecase/webapi -run '^TestBudgetPrivacy' -count=1
```

Observed failure: 49 privacy assertion matches in `budget_privacy_test.go`. Synthetic private sentinels and request URLs appeared in OpenRouter transport/body-read returned errors, Gemini SDK/model-decode errors or logs, and wrapper errors/logs (including actual Gemini fallback failures). These were behavior failures, not build failures.

GREEN: the exact RED command passed after implementation (`ok .../internal/usecase/webapi 0.019s`). After adding alternate pipeline/retry cases and strengthening assertions, the same command passed again (`0.022s`).

Implementation boundaries:

- Budget errors contain only locally constructed classes and safe HTTP status/provider code. Raw errors are discarded, not retained anywhere in the error chain; only `context.Canceled` or `context.DeadlineExceeded` sentinel identity is retained for `errors.Is`.
- Gemini budget SDK failures return a fixed SDK class. Malformed budget JSON logs a fixed class without response text or decoder details.
- The budget fallback wrapper sanitizes both providers independently, including unknown providers and untrusted wrappers around otherwise safe errors. A successful fallback still returns its result. Joined sanitized failures preserve context identity from either failed provider.
- OpenRouter budget transport/body-read failures keep all four existing retry attempts and delays. Budget-only success logging no longer emits token counts or response-length metadata. The transaction batch caller uses the legacy mode of the shared helper, preserving the current candidate's behavior; single-SMS and transaction-model extraction are unchanged by this second correction.

Regression coverage uses synthetic data and intercepted HTTP only, including the real genai SDK and `GeminiService` as fallback. Cases cover transport/body-read exhaustion and later retry success, Gemini SDK rejection, malformed model JSON and type errors, fallback absence/failure/success, a provider violating the error contract, and context cancellation/deadline errors. `testing/synctest` advances backoffs without wall-clock sleeps. Assertions check returned errors and captured provider/wrapper logs for synthetic content and provider/request URLs, check retry/fallback call counts, and ensure raw I/O causes are absent from the error chain.

Validation after the second correction:

- `go test ./internal/usecase/webapi -count=1`: passed (`0.061s`).
- `go test ./internal/usecase -run '^(TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1`: passed (`0.064s`).
- `go test -race ./internal/usecase/webapi ./internal/usecase -run '^(TestBudgetPrivacy|TestOpenRouterBatchBudget|TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1`: passed (webapi `1.126s`, usecase `1.624s`), no race reports.
- `go vet ./internal/usecase/webapi ./internal/usecase`: exit 0, no findings.
- `gofmt -w internal/usecase/webapi/ai_service.go internal/usecase/webapi/gemini_service.go internal/usecase/webapi/openrouter_service.go internal/usecase/webapi/budget_privacy_test.go internal/usecase/webapi/budget_pipeline_privacy_test.go`: exit 0. Final formatting and whitespace checks are recorded below.

Final checks:

```sh
gofmt -l internal/usecase/webapi/ai_service.go internal/usecase/webapi/gemini_service.go internal/usecase/webapi/openrouter_service.go internal/usecase/webapi/openrouter_batch_test.go internal/usecase/webapi/budget_privacy_test.go internal/usecase/webapi/budget_pipeline_privacy_test.go internal/usecase/bank_notification_pattern.go internal/usecase/sms_suggestion_processing_test.go internal/usecase/sms_suggestion_job_test.go
```

Exit 0, no unformatted paths. `git diff --check`: exit 0, no whitespace errors.

### Final blocker — context identity at the usecase/job boundary

Independent review found that the usecase discarded sanitized chunk errors and created a fresh generic error when every attempted chunk failed. The provider wrapper preserved context identity, but `errors.Is` could no longer find it at the usecase boundary.

RED command (new boundary tests, before implementation):

```sh
go test ./internal/usecase -run '^TestProcessSMSBatchForSuggestions_(AllChunkContextFailures|PartialSuccessWithContextFailure)$' -count=1
```

Observed exit 1: `AllChunkContextFailures/canceled` and `/deadline` returned the generic all-failed error without the required sentinel identity. Mixed context failures and context/non-context failures also lost identity. Partial-success cases already passed in both chunk orders.

GREEN: the exact RED command passed after implementation (`ok .../internal/usecase 0.078s`).

The usecase retains only `context.Canceled` and `context.DeadlineExceeded` sentinels, never a provider error or its chain. When no chunk succeeds, it wraps the retained sentinel identities with the existing sanitized all-failed message. Non-context failures keep the exact existing generic error. A successful chunk still wins regardless of the order of a provider context failure; the existing processing loop and incoming-context behavior are unchanged.

Triangulation covers both sentinel identities together, context/non-context failures in either order, both partial-success orders for each sentinel, raw-error chain exclusion, aggregate-count logs, and exact sanitized failed-job errors with no result JSON. Tests intercept HTTP using synthetic wrapped provider errors and virtual backoff time; no live requests or runtime data are used.

Validation:

- `go test ./internal/usecase -run '^(TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1`: passed (`0.111s`).
- `go test ./internal/usecase/webapi -count=1`: passed (`0.019s`).
- `go test -race ./internal/usecase/webapi ./internal/usecase -run '^(TestBudgetPrivacy|TestOpenRouterBatchBudget|TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1`: passed (webapi `1.075s`, usecase `2.070s`), no race reports.
- `go vet ./internal/usecase/webapi ./internal/usecase`: exit 0, no findings.
- `gofmt -w internal/usecase/bank_notification_pattern.go internal/usecase/sms_suggestion_processing_test.go internal/usecase/sms_suggestion_job_test.go`: exit 0. Subsequent `gofmt -l` on these same three paths emitted no paths; `git diff --check` passed.

Only the four authorized usecase/test/ODD paths were changed for this blocker. Pre-existing provider changes and unrelated files were preserved. No commit was created.

## Strict ZDR routing extension

Approved scope: route SMS budget extraction through `deepseek/deepseek-v4.1-flash` with mandatory OpenRouter `provider.zdr=true`, fail closed when no eligible route exists, and prevent this strict path from bypassing ZDR through direct Gemini fallback. Other AI paths and fallbacks remain unchanged.

- [x] Add RED request-contract tests for the exact model and `provider.zdr=true` on every retry.
- [x] Add RED fallback regression proving the SMS budget path never invokes Gemini/non-ZDR fallback.
- [x] Implement the smallest budget-only OpenRouter request extension and strict fallback behavior.
- [x] Run focused tests, race checks, vet, formatting, and diff checks.

Constraints: no live inference, provider-account changes, deployment, commit, push, issue, PR, or merge. Synthetic fixtures only.

### Strict ZDR — observed RED/GREEN evidence

The initially attempted offline command stopped before tests: `go.mod requires go >= 1.26.0 (running go 1.22.2; GOTOOLCHAIN=local)`. The installed cached Go 1.26.9 toolchain was then selected without downloading or changing dependencies. All Go validation below ran with this shell setup:

```sh
export PATH=/root/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.9.linux-amd64/bin:$PATH
```

RED, after adding/updating tests and before changing implementation:

```sh
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go test ./internal/usecase/webapi -run '^(TestOpenRouterBatchBudget|TestBudgetPrivacy)' -count=1
```

Observed behavior failure: the test runner reported 39 assertion matches across the three test files. Every captured budget attempt used `openrouter/free` with no provider object; configured non-ZDR/Gemini fallbacks were invoked, including after a no-eligible-route rejection; Gemini-as-primary accepted budget extraction instead of returning unavailable. No live HTTP was used. An earlier run also exposed the pre-existing nil-primary panic; nil/typed-nil guard cases were added after GREEN as alternate-case coverage.

GREEN: the exact offline RED command passed after implementation (`ok .../internal/usecase/webapi 0.014s`). After adding unrelated-route and nil-primary alternate cases and formatting, it passed again (`0.017s`).

Implementation and regression boundaries:

- Only budget-mode requests select `deepseek/deepseek-v4.1-flash` and the nested object `provider: {"zdr": true}`. The request is marshaled once before retries, so policy cannot be omitted or downgraded between attempts. The shared default remains `openrouter/free`; its requests omit `provider`.
- The budget wrapper accepts only a non-nil `*OpenRouterService`. Gemini promoted to primary, unknown providers, unavailable providers, and nil providers return `ErrAIServiceUnavailable` without a provider call. OpenRouter failures remain sanitized and retain context sentinel identity; no fallback is invoked, even if a configured Gemini fallback could succeed. This supersedes the earlier budget-fallback-success behavior recorded above.
- Mocked HTTP covers HTTP/envelope rate limits, transport/read exhaustion, HTTP/envelope no-eligible-ZDR-route rejection, and legitimate zero-expense success. Every attempt asserts exact model/provider, unchanged budget prompt/body/date text and incoming context, plus cumulative virtual retry delays of 0/3/11/26 seconds. Existing retry-success, cancellation, status/error-class, log/error sanitization, partial-success, and usecase context tests remain green.
- Single-SMS and transaction-batch tests assert the unchanged free model with no provider restriction, successful configured Gemini fallback, successful Gemini-as-primary extraction, and preserved parsed date semantics. Chat/statement implementations were not edited. Direct Gemini budget tests remain as isolated privacy tests, not as a reachable strict-budget fallback route.

Observed validation (all personally run):

```sh
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go test -race ./internal/usecase/webapi ./internal/usecase -run '^(TestBudgetPrivacy|TestOpenRouterBatchBudget|TestProcessSMSBatchForSuggestions_|TestSMSBatchSuggestionJob_|TestLikelyBankTransactionSMS_|TestStartSMSBatchSuggestionJob_)' -count=1
```

Passed: webapi `1.084s`, usecase `1.943s`, no race reports.

```sh
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go vet ./internal/usecase/webapi ./internal/usecase
```

Exit 0, no findings.

```sh
gofmt -w internal/usecase/webapi/openrouter_service.go internal/usecase/webapi/openrouter_batch_test.go internal/usecase/webapi/budget_pipeline_privacy_test.go internal/usecase/webapi/ai_service.go internal/usecase/webapi/budget_privacy_test.go
```

Exit 0. `git diff --check` also passed after the documentation update, with no whitespace errors.

Independent verification repeated the focused webapi tests (`0.018s`), SMS usecase/job tests (`0.083s`), race checks (webapi `1.074s`, usecase `1.853s`), vet, formatting, and diff checks with the cached Go 1.26.9 binary; all passed. Native assessment remained unavailable because the candidate includes intended untracked test/document files, so this independent verification is the final verification record.

Limitations: deterministic mocks verify request policy and fail-closed handling, not current model availability, provider ZDR eligibility, or remote retention guarantees. No live provider calls, account-policy changes, dependency mutation, deployment, staging, commit, push, or PR actions were performed. The existing candidate remains uncommitted; unrelated candidate files were preserved.

## Delivery

- Branch: `fix/sms-empty-result`
- Failure-semantics work unit: `e46df9ab5e9b5f6f8e33318865557f5f8626f893` (`fix(ai): surface SMS suggestion processing failures`).
- Strict-ZDR/privacy work unit: `ef5329d8bbd063142dc2b757ac1572ab6310d2d8` (`fix(ai): enforce ZDR routing for SMS budgets`).
- Both commits are local only; push, pull request, merge, and deployment remain unauthorized.
