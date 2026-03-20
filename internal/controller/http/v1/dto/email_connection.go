package dto

import "time"

// GmailAuthorizeResponse URL para abrir en el navegador / WebView.
type GmailAuthorizeResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// EmailConnectionStatus estado de la vinculación correo.
type EmailConnectionStatus struct {
	Connected    bool       `json:"connected"`
	Provider     string     `json:"provider,omitempty"`
	EmailAddress string     `json:"email_address,omitempty"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
}

// GmailSyncResponse resumen tras sync manual o interno.
type GmailSyncResponse struct {
	MessagesExamined int `json:"messages_examined"`
	MessagesSkipped  int `json:"messages_skipped"`
	ProcessedWithAI  int `json:"processed_with_ai"`
	Errors           int `json:"errors"`
}
