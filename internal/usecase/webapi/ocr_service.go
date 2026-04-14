package webapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOCRProviderURL = "https://api.ocr.space/parse/image"
)

type OCRService struct {
	apiKey      string
	providerURL string
	client      *http.Client
}

type OCRExtractor interface {
	ExtractText(ctx context.Context, filename string, content []byte) (string, error)
}

type ocrProviderResponse struct {
	IsErroredOnProcessing bool `json:"IsErroredOnProcessing"`
	ParsedResults         []struct {
		ParsedText string `json:"ParsedText"`
	} `json:"ParsedResults"`
	ErrorMessage interface{} `json:"ErrorMessage"`
}

func NewOCRServiceFromEnv() (*OCRService, error) {
	apiKey := strings.TrimSpace(os.Getenv("OCR_API_KEY"))
	if apiKey == "" {
		return nil, nil
	}
	providerURL := strings.TrimSpace(os.Getenv("OCR_PROVIDER_URL"))
	if providerURL == "" {
		providerURL = defaultOCRProviderURL
	}
	return &OCRService{
		apiKey:      apiKey,
		providerURL: providerURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func (s *OCRService) ExtractText(ctx context.Context, filename string, content []byte) (string, error) {
	if s == nil || s.apiKey == "" {
		return "", fmt.Errorf("ocr provider not configured")
	}
	if len(content) == 0 {
		return "", fmt.Errorf("empty file content")
	}

	var lastErr error
	backoffs := []time.Duration{0, 2 * time.Second, 5 * time.Second}
	for _, wait := range backoffs {
		if wait > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}

		text, err := s.extractTextAttempt(ctx, filename, content)
		if err == nil {
			return text, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("ocr extraction failed after retries: %w", lastErr)
}

func (s *OCRService) extractTextAttempt(ctx context.Context, filename string, content []byte) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("apikey", s.apiKey)
	_ = writer.WriteField("OCREngine", "2")
	_ = writer.WriteField("isTable", "true")
	_ = writer.WriteField("scale", "true")
	_ = writer.WriteField("language", "spa")

	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := fileWriter.Write(content); err != nil {
		return "", fmt.Errorf("write form file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.providerURL, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ocr request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return "", fmt.Errorf("retryable ocr status %d: %s", resp.StatusCode, string(respBody))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ocr status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed ocrProviderResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("parse ocr response: %w", err)
	}
	if parsed.IsErroredOnProcessing {
		return "", fmt.Errorf("ocr provider processing error: %v", parsed.ErrorMessage)
	}

	textParts := make([]string, 0, len(parsed.ParsedResults))
	for _, result := range parsed.ParsedResults {
		if strings.TrimSpace(result.ParsedText) != "" {
			textParts = append(textParts, result.ParsedText)
		}
	}
	if len(textParts) == 0 {
		return "", fmt.Errorf("ocr provider returned empty parsed text")
	}

	return strings.Join(textParts, "\n"), nil
}

func (s *OCRService) String() string {
	if s == nil {
		return "OCRService(nil)"
	}
	return "OCRService(url=" + s.providerURL + ", configured=" + strconv.FormatBool(s.apiKey != "") + ")"
}
