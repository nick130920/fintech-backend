package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/logger"
	"github.com/nick130920/fintech-backend/pkg/oauthstate"
	"github.com/nick130920/fintech-backend/pkg/tokenencrypt"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const emailProviderGmail = "gmail"

// EmailGmailUseCase OAuth Gmail + sync + IA vía BankNotificationPatternUseCase.
type EmailGmailUseCase struct {
	cfg           *configs.Config
	connRepo      repo.UserEmailConnectionRepo
	procRepo      repo.ProcessedEmailMessageRepo
	enc           *tokenencrypt.Encrypter
	bankPatternUC *BankNotificationPatternUseCase
	oauth2Cfg     *oauth2.Config
	stateSecret   string
	log           zerolog.Logger
}

// NewEmailGmailUseCase constructor.
func NewEmailGmailUseCase(
	cfg *configs.Config,
	connRepo repo.UserEmailConnectionRepo,
	procRepo repo.ProcessedEmailMessageRepo,
	bankPatternUC *BankNotificationPatternUseCase,
) (*EmailGmailUseCase, error) {
	g := cfg.External.Gmail
	gmailOn := g.ClientID != "" && g.ClientSecret != "" && g.RedirectURL != ""

	var enc *tokenencrypt.Encrypter
	var err error
	if gmailOn {
		if strings.TrimSpace(g.TokenEncryptionKey) == "" {
			return nil, errors.New(
				"email gmail: TOKEN_ENCRYPTION_KEY es obligatoria cuando Gmail OAuth está configurada (no se usa JWT como clave de cifrado)",
			)
		}
		if strings.TrimSpace(g.OAuthStateSecret) == "" {
			return nil, errors.New(
				"email gmail: OAUTH_STATE_SECRET es obligatoria cuando Gmail OAuth está configurada (no se usa JWT para firmar state)",
			)
		}
		enc, err = tokenencrypt.New(g.TokenEncryptionKey, "")
		if err != nil {
			return nil, fmt.Errorf("email gmail: token encrypt: %w", err)
		}
	} else {
		enc, err = tokenencrypt.New(g.TokenEncryptionKey, cfg.JWT.SecretKey)
		if err != nil {
			return nil, fmt.Errorf("email gmail: token encrypt: %w", err)
		}
	}

	stateSecret := g.OAuthStateSecret
	if !gmailOn && strings.TrimSpace(stateSecret) == "" {
		stateSecret = cfg.JWT.SecretKey
	}
	o2 := &oauth2.Config{
		ClientID:     g.ClientID,
		ClientSecret: g.ClientSecret,
		RedirectURL:  g.RedirectURL,
		Scopes:       []string{gmail.GmailReadonlyScope},
		Endpoint:     google.Endpoint,
	}
	return &EmailGmailUseCase{
		cfg:           cfg,
		connRepo:      connRepo,
		procRepo:      procRepo,
		enc:           enc,
		bankPatternUC: bankPatternUC,
		oauth2Cfg:     o2,
		stateSecret:   stateSecret,
		log:           logger.Get(),
	}, nil
}

// IsGmailConfigured true si hay client id y redirect (para habilitar rutas).
func (uc *EmailGmailUseCase) IsGmailConfigured() bool {
	g := uc.cfg.External.Gmail
	return g.ClientID != "" && g.RedirectURL != "" && g.ClientSecret != ""
}

// BuildGmailAuthorizeURL genera URL OAuth para el usuario autenticado.
func (uc *EmailGmailUseCase) BuildGmailAuthorizeURL(userID uint) (*dto.GmailAuthorizeResponse, error) {
	if !uc.IsGmailConfigured() {
		return nil, errors.New("gmail oauth no configurado en el servidor")
	}
	state, err := oauthstate.Sign(userID, 15*time.Minute, uc.stateSecret)
	if err != nil {
		return nil, err
	}
	authURL := uc.oauth2Cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	return &dto.GmailAuthorizeResponse{AuthURL: authURL, State: state}, nil
}

// HandleGmailCallback intercambia code, guarda tokens cifrados.
func (uc *EmailGmailUseCase) HandleGmailCallback(ctx context.Context, state, code string) error {
	uid, err := oauthstate.Verify(state, uc.stateSecret)
	if err != nil {
		return fmt.Errorf("state inválido: %w", err)
	}
	tok, err := uc.oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("intercambio oauth: %w", err)
	}
	if tok.RefreshToken == "" {
		uc.log.Warn().Uint("user_id", uid).Msg("gmail oauth sin refresh_token; revocar app en Google y reconectar si falla el sync")
	}

	client := uc.oauth2Cfg.Client(ctx, tok)
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("cliente gmail: %w", err)
	}
	prof, err := svc.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("perfil gmail: %w", err)
	}

	rtEnc, err := uc.enc.Encrypt(tok.RefreshToken)
	if err != nil {
		return err
	}
	atEnc, err := uc.enc.Encrypt(tok.AccessToken)
	if err != nil {
		return err
	}
	var exp *time.Time
	if !tok.Expiry.IsZero() {
		t := tok.Expiry.UTC()
		exp = &t
	}

	hist := fmt.Sprintf("%d", prof.HistoryId)
	conn := &entity.UserEmailConnection{
		UserID:          uid,
		Provider:        emailProviderGmail,
		EmailAddress:    prof.EmailAddress,
		RefreshTokenEnc: rtEnc,
		AccessTokenEnc:  atEnc,
		AccessExpiresAt: exp,
		LastHistoryID:   hist,
		RevokedAt:       nil,
	}
	return uc.connRepo.CreateOrUpdate(conn)
}

// GetEmailStatus resumen para la app.
func (uc *EmailGmailUseCase) GetEmailStatus(userID uint) (*dto.EmailConnectionStatus, error) {
	c, err := uc.connRepo.GetByUserAndProvider(userID, emailProviderGmail)
	if err != nil {
		return nil, err
	}
	if c == nil || c.RevokedAt != nil {
		return &dto.EmailConnectionStatus{Connected: false}, nil
	}
	return &dto.EmailConnectionStatus{
		Connected:    true,
		Provider:     c.Provider,
		EmailAddress: c.EmailAddress,
		LastSyncedAt: c.LastSyncedAt,
	}, nil
}

// DisconnectGmail revoca token en Google (best-effort) y marca conexión local.
func (uc *EmailGmailUseCase) DisconnectGmail(ctx context.Context, userID uint) error {
	c, err := uc.connRepo.GetByUserAndProvider(userID, emailProviderGmail)
	if err != nil {
		return err
	}
	if c == nil {
		return nil
	}
	rt, err := uc.enc.Decrypt(c.RefreshTokenEnc)
	if err == nil && rt != "" {
		v := url.Values{}
		v.Set("token", rt)
		rq, errRq := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/revoke", strings.NewReader(v.Encode()))
		if errRq == nil {
			rq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, errDo := http.DefaultClient.Do(rq)
			if errDo == nil && resp != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		}
	}
	return uc.connRepo.SoftRevoke(userID, emailProviderGmail)
}

// gmailHTTPClient obtiene cliente HTTP con token válido.
func (uc *EmailGmailUseCase) gmailHTTPClient(ctx context.Context, c *entity.UserEmailConnection) (*http.Client, error) {
	rt, err := uc.enc.Decrypt(c.RefreshTokenEnc)
	if err != nil {
		return nil, err
	}
	if rt == "" {
		return nil, errors.New("sin refresh_token gmail")
	}
	tok := &oauth2.Token{RefreshToken: rt}
	if c.AccessTokenEnc != "" && c.AccessExpiresAt != nil && c.AccessExpiresAt.After(time.Now().Add(2*time.Minute)) {
		if at, err := uc.enc.Decrypt(c.AccessTokenEnc); err == nil && at != "" {
			tok.AccessToken = at
			tok.Expiry = *c.AccessExpiresAt
		}
	}
	return uc.oauth2Cfg.Client(ctx, tok), nil
}

// persistTokenSnapshot guarda access token tras refresh implícito.
func (uc *EmailGmailUseCase) persistTokenSnapshot(ctx context.Context, c *entity.UserEmailConnection, client *http.Client) {
	// oauth2.Transport no expone token fácilmente; omitimos persistencia fina del access token en v1.
	_ = ctx
	_ = c
	_ = client
}

// SyncGmailForUser sincroniza buzón y procesa con IA.
func (uc *EmailGmailUseCase) SyncGmailForUser(ctx context.Context, userID uint) (*dto.GmailSyncResponse, error) {
	out := &dto.GmailSyncResponse{}
	if !uc.IsGmailConfigured() {
		return out, errors.New("gmail no configurado")
	}
	c, err := uc.connRepo.GetByUserAndProvider(userID, emailProviderGmail)
	if err != nil {
		return nil, err
	}
	if c == nil || c.RevokedAt != nil {
		return nil, errors.New("gmail no conectado")
	}
	httpClient, err := uc.gmailHTTPClient(ctx, c)
	if err != nil {
		return nil, err
	}
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	uc.persistTokenSnapshot(ctx, c, httpClient)

	prof, err := svc.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	query := uc.cfg.External.Gmail.DefaultListQuery
	if strings.TrimSpace(query) == "" {
		query = "newer_than:90d"
	}

	var messageIDs []string
	if c.LastHistoryID != "" {
		var start uint64
		_, _ = fmt.Sscanf(c.LastHistoryID, "%d", &start)
		if start > 0 {
			hist, errH := svc.Users.History.List("me").StartHistoryId(start).Context(ctx).Do()
			if errH == nil && hist != nil {
				seen := make(map[string]bool)
				for _, h := range hist.History {
					for _, added := range h.MessagesAdded {
						if added.Message != nil && added.Message.Id != "" && !seen[added.Message.Id] {
							seen[added.Message.Id] = true
							messageIDs = append(messageIDs, added.Message.Id)
						}
					}
				}
			}
		}
	}
	if len(messageIDs) > 25 {
		messageIDs = messageIDs[:25]
	}
	if len(messageIDs) == 0 {
		list, errL := svc.Users.Messages.List("me").Q(query).MaxResults(25).Context(ctx).Do()
		if errL != nil {
			return nil, errL
		}
		for _, m := range list.Messages {
			if m != nil && m.Id != "" {
				messageIDs = append(messageIDs, m.Id)
			}
		}
	}

	htmlStrip := regexp.MustCompile(`(?s)<[^>]+>`)

	for _, mid := range messageIDs {
		out.MessagesExamined++
		exists, errE := uc.procRepo.Exists(userID, emailProviderGmail, mid)
		if errE != nil {
			out.Errors++
			continue
		}
		if exists {
			out.MessagesSkipped++
			continue
		}

		msg, errG := svc.Users.Messages.Get("me", mid).Format("full").Context(ctx).Do()
		if errG != nil {
			out.Errors++
			continue
		}
		subj, from := gmailHeaders(msg)
		body := extractGmailBody(msg)
		combined := strings.TrimSpace(from + " " + subj + " " + body)
		if combined == "" {
			_ = uc.procRepo.Create(&entity.ProcessedEmailMessage{UserID: userID, Provider: emailProviderGmail, ProviderMessageID: mid})
			out.MessagesSkipped++
			continue
		}
		if !likelyBankTransactionSMS(combined) {
			_ = uc.procRepo.Create(&entity.ProcessedEmailMessage{UserID: userID, Provider: emailProviderGmail, ProviderMessageID: mid})
			out.MessagesSkipped++
			continue
		}

		plain := htmlStrip.ReplaceAllString(body, " ")
		plain = strings.Join(strings.Fields(plain), " ")
		text := strings.TrimSpace(fmt.Sprintf("De: %s\nAsunto: %s\n%s", from, subj, plain))
		if len(text) > 8000 {
			text = text[:8000]
		}

		_, errAI := uc.bankPatternUC.ProcessSMSWithAI(ctx, userID, text)
		if errAI != nil {
			uc.log.Warn().Err(errAI).Str("msg_id", mid).Msg("gmail sync IA error")
			out.Errors++
		} else {
			out.ProcessedWithAI++
		}
		if errP := uc.procRepo.Create(&entity.ProcessedEmailMessage{UserID: userID, Provider: emailProviderGmail, ProviderMessageID: mid}); errP != nil {
			uc.log.Warn().Err(errP).Str("msg_id", mid).Msg("processed_email insert")
		}
	}

	now := time.Now().UTC()
	c.LastHistoryID = fmt.Sprintf("%d", prof.HistoryId)
	c.LastSyncedAt = &now
	if err := uc.connRepo.CreateOrUpdate(c); err != nil {
		uc.log.Warn().Err(err).Msg("actualizar conexión gmail tras sync")
	}
	return out, nil
}

// SyncAllGmailConnections worker: todas las cuentas activas.
func (uc *EmailGmailUseCase) SyncAllGmailConnections(ctx context.Context) {
	if !uc.IsGmailConfigured() {
		return
	}
	list, err := uc.connRepo.ListActiveByProvider(emailProviderGmail)
	if err != nil {
		uc.log.Warn().Err(err).Msg("listar conexiones gmail")
		return
	}
	for i := range list {
		conn := list[i]
		subCtx, cancel := context.WithTimeout(ctx, 8*time.Minute)
		_, errS := uc.SyncGmailForUser(subCtx, conn.UserID)
		cancel()
		if errS != nil {
			uc.log.Warn().Err(errS).Uint("user_id", conn.UserID).Msg("sync gmail usuario")
		}
	}
}

func gmailHeaders(msg *gmail.Message) (subject, from string) {
	if msg.Payload == nil || msg.Payload.Headers == nil {
		return "", ""
	}
	for _, h := range msg.Payload.Headers {
		if h == nil {
			continue
		}
		switch strings.ToLower(h.Name) {
		case "subject":
			subject = h.Value
		case "from":
			from = h.Value
		}
	}
	return subject, from
}

func extractGmailBody(msg *gmail.Message) string {
	if msg.Payload == nil {
		return ""
	}
	var plain, html string
	var walk func(*gmail.MessagePart)
	walk = func(p *gmail.MessagePart) {
		if p == nil {
			return
		}
		if p.Body != nil && p.Body.Data != "" {
			dec := decodeGmailBodyData(p.Body.Data)
			switch p.MimeType {
			case "text/plain":
				if dec != "" {
					plain = dec
				}
			case "text/html":
				if dec != "" {
					html = dec
				}
			}
		}
		for _, ch := range p.Parts {
			walk(ch)
		}
	}
	walk(msg.Payload)
	if plain != "" {
		return plain
	}
	return html
}

func decodeGmailBodyData(s string) string {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(b)
}
