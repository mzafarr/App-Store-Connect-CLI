package googleplay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	maxResponseBodyBytes  int64 = 1 << 20
	defaultRequestTimeout       = 30 * time.Second
	timeoutEnvVar               = "GOOGLE_PLAY_TIMEOUT"
	timeoutSecondsEnvVar        = "GOOGLE_PLAY_TIMEOUT_SECONDS"
)

// Option configures a Google Play client.
type Option func(*Client)

// Client is a minimal Android Publisher API client.
type Client struct {
	account    ServiceAccount
	httpClient *http.Client
	baseURL    string
	tokenURL   string
	now        func() time.Time
}

// WithHTTPClient overrides the HTTP client used for OAuth and API requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithBaseURL overrides the Android Publisher API base URL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

// WithTokenURL overrides the OAuth token URL.
func WithTokenURL(tokenURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(tokenURL) != "" {
			c.tokenURL = strings.TrimSpace(tokenURL)
		}
	}
}

// WithNow overrides the current-time function (tests).
func WithNow(nowFn func() time.Time) Option {
	return func(c *Client) {
		if nowFn != nil {
			c.now = nowFn
		}
	}
}

// NewClientFromEnv creates a client using environment-based credentials.
func NewClientFromEnv(opts ...Option) (*Client, error) {
	account, err := LoadServiceAccountFromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(account, opts...)
}

// NewClientFromProfile resolves credentials from env/config using selected profile.
func NewClientFromProfile(profile string, opts ...Option) (*Client, CredentialSource, error) {
	account, source, err := LoadServiceAccount(profile)
	if err != nil {
		return nil, CredentialSource{}, err
	}
	client, err := NewClient(account, opts...)
	if err != nil {
		return nil, CredentialSource{}, err
	}
	return client, source, nil
}

// NewClient creates a client with an explicit service account.
func NewClient(account ServiceAccount, opts ...Option) (*Client, error) {
	account.ClientEmail = strings.TrimSpace(account.ClientEmail)
	account.PrivateKey = normalizeMultilineSecret(account.PrivateKey)
	account.TokenURI = strings.TrimSpace(account.TokenURI)
	if account.TokenURI == "" {
		account.TokenURI = defaultTokenURL
	}

	if account.ClientEmail == "" {
		return nil, fmt.Errorf("googleplay auth: service account is missing client_email")
	}
	if strings.TrimSpace(account.PrivateKey) == "" {
		return nil, fmt.Errorf("googleplay auth: service account is missing private_key")
	}

	client := &Client{
		account:    account,
		httpClient: &http.Client{Timeout: resolveRequestTimeout()},
		baseURL:    resolveAPIBaseURL(account.APIBaseURL),
		tokenURL:   resolveTokenURL(account.TokenURI),
		now:        time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}

	return client, nil
}

// ListUsers lists users on a Play developer account.
func (c *Client) ListUsers(ctx context.Context, developer string, pageSize int, pageToken string) (ListUsersResponse, error) {
	parent, err := normalizeDeveloperParent(developer)
	if err != nil {
		return ListUsersResponse{}, err
	}

	query := url.Values{}
	if pageSize != 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	if token := strings.TrimSpace(pageToken); token != "" {
		query.Set("pageToken", token)
	}

	var out ListUsersResponse
	endpoint := fmt.Sprintf("/androidpublisher/v3/%s/users", parent)
	if err := c.doJSON(ctx, http.MethodGet, endpoint, query, nil, &out); err != nil {
		return ListUsersResponse{}, err
	}
	return out, nil
}

// CreateEdit creates a new edit for a package.
func (c *Client) CreateEdit(ctx context.Context, packageName string) (AppEdit, error) {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return AppEdit{}, fmt.Errorf("package name is required")
	}

	var out AppEdit
	endpoint := fmt.Sprintf("/androidpublisher/v3/applications/%s/edits", url.PathEscape(packageName))
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, nil, &out); err != nil {
		return AppEdit{}, err
	}
	return out, nil
}

// CommitEdit commits an existing edit.
func (c *Client) CommitEdit(ctx context.Context, packageName, editID string, changesNotSentForReview *bool) (AppEdit, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	if packageName == "" {
		return AppEdit{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return AppEdit{}, fmt.Errorf("edit ID is required")
	}

	query := url.Values{}
	if changesNotSentForReview != nil {
		query.Set("changesNotSentForReview", fmt.Sprintf("%t", *changesNotSentForReview))
	}

	var out AppEdit
	endpoint := fmt.Sprintf(
		"/androidpublisher/v3/applications/%s/edits/%s:commit",
		url.PathEscape(packageName),
		url.PathEscape(editID),
	)
	if err := c.doJSON(ctx, http.MethodPost, endpoint, query, nil, &out); err != nil {
		return AppEdit{}, err
	}
	return out, nil
}

// ListTracks lists tracks for an edit.
func (c *Client) ListTracks(ctx context.Context, packageName, editID string) (TracksListResponse, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	if packageName == "" {
		return TracksListResponse{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return TracksListResponse{}, fmt.Errorf("edit ID is required")
	}

	var out TracksListResponse
	endpoint := fmt.Sprintf(
		"/androidpublisher/v3/applications/%s/edits/%s/tracks",
		url.PathEscape(packageName),
		url.PathEscape(editID),
	)
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return TracksListResponse{}, err
	}
	return out, nil
}

// ListListings lists localized store listings for an edit.
func (c *Client) ListListings(ctx context.Context, packageName, editID string) (ListingsListResponse, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	if packageName == "" {
		return ListingsListResponse{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return ListingsListResponse{}, fmt.Errorf("edit ID is required")
	}

	var out ListingsListResponse
	endpoint := fmt.Sprintf(
		"/androidpublisher/v3/applications/%s/edits/%s/listings",
		url.PathEscape(packageName),
		url.PathEscape(editID),
	)
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return ListingsListResponse{}, err
	}
	return out, nil
}

// GetListing gets a localized store listing for an edit and language.
func (c *Client) GetListing(ctx context.Context, packageName, editID, language string) (Listing, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	language = strings.TrimSpace(language)
	if packageName == "" {
		return Listing{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return Listing{}, fmt.Errorf("edit ID is required")
	}
	if language == "" {
		return Listing{}, fmt.Errorf("language is required")
	}

	var out Listing
	endpoint := fmt.Sprintf(
		"/androidpublisher/v3/applications/%s/edits/%s/listings/%s",
		url.PathEscape(packageName),
		url.PathEscape(editID),
		url.PathEscape(language),
	)
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return Listing{}, err
	}
	return out, nil
}

// PatchListing patches a localized store listing for an edit and language.
func (c *Client) PatchListing(ctx context.Context, packageName, editID, language string, payload Listing) (Listing, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	language = strings.TrimSpace(language)
	if packageName == "" {
		return Listing{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return Listing{}, fmt.Errorf("edit ID is required")
	}
	if language == "" {
		return Listing{}, fmt.Errorf("language is required")
	}

	var out Listing
	endpoint := fmt.Sprintf(
		"/androidpublisher/v3/applications/%s/edits/%s/listings/%s",
		url.PathEscape(packageName),
		url.PathEscape(editID),
		url.PathEscape(language),
	)
	if err := c.doJSON(ctx, http.MethodPatch, endpoint, nil, payload, &out); err != nil {
		return Listing{}, err
	}
	return out, nil
}

// UpdateTrack updates a specific track for an edit.
func (c *Client) UpdateTrack(ctx context.Context, packageName, editID, track string, payload UpdateTrackRequest) (Track, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	track = strings.TrimSpace(track)
	if packageName == "" {
		return Track{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return Track{}, fmt.Errorf("edit ID is required")
	}
	if track == "" {
		return Track{}, fmt.Errorf("track is required")
	}

	var out Track
	endpoint := fmt.Sprintf(
		"/androidpublisher/v3/applications/%s/edits/%s/tracks/%s",
		url.PathEscape(packageName),
		url.PathEscape(editID),
		url.PathEscape(track),
	)
	if err := c.doJSON(ctx, http.MethodPut, endpoint, nil, payload, &out); err != nil {
		return Track{}, err
	}
	return out, nil
}

// UploadBundle uploads an Android App Bundle (.aab) to an edit.
func (c *Client) UploadBundle(ctx context.Context, packageName, editID, bundlePath string) (Bundle, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	bundlePath = strings.TrimSpace(bundlePath)

	if packageName == "" {
		return Bundle{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return Bundle{}, fmt.Errorf("edit ID is required")
	}
	if bundlePath == "" {
		return Bundle{}, fmt.Errorf("bundle path is required")
	}

	var bundle Bundle
	if err := c.uploadBinary(
		ctx,
		packageName,
		editID,
		"bundles",
		bundlePath,
		&bundle,
	); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

// UploadAPK uploads an APK file to an edit.
func (c *Client) UploadAPK(ctx context.Context, packageName, editID, apkPath string) (APK, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	apkPath = strings.TrimSpace(apkPath)

	if packageName == "" {
		return APK{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return APK{}, fmt.Errorf("edit ID is required")
	}
	if apkPath == "" {
		return APK{}, fmt.Errorf("apk path is required")
	}

	var apk APK
	if err := c.uploadBinary(
		ctx,
		packageName,
		editID,
		"apks",
		apkPath,
		&apk,
	); err != nil {
		return APK{}, err
	}
	return apk, nil
}

// GetTrack retrieves track state for an existing edit.
func (c *Client) GetTrack(ctx context.Context, packageName, editID, track string) (Track, error) {
	packageName = strings.TrimSpace(packageName)
	editID = strings.TrimSpace(editID)
	track = strings.TrimSpace(track)

	if packageName == "" {
		return Track{}, fmt.Errorf("package name is required")
	}
	if editID == "" {
		return Track{}, fmt.Errorf("edit ID is required")
	}
	if track == "" {
		return Track{}, fmt.Errorf("track is required")
	}

	var out Track
	endpoint := fmt.Sprintf(
		"/androidpublisher/v3/applications/%s/edits/%s/tracks/%s",
		url.PathEscape(packageName),
		url.PathEscape(editID),
		url.PathEscape(track),
	)
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return Track{}, err
	}
	return out, nil
}

func (c *Client) uploadBinary(ctx context.Context, packageName, editID, resourcePath, filePath string, out any) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("googleplay: open %s file: %w", resourcePath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("googleplay: stat %s file: %w", resourcePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("googleplay: %s path must be a file", resourcePath)
	}

	token, err := c.token(ctx)
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("uploadType", "media")
	endpoint := fmt.Sprintf(
		"/upload/androidpublisher/v3/applications/%s/edits/%s/%s",
		url.PathEscape(packageName),
		url.PathEscape(editID),
		url.PathEscape(resourcePath),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpointURL(endpoint, query), file)
	if err != nil {
		return fmt.Errorf("googleplay: create upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("googleplay: upload %s failed: %w", resourcePath, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return fmt.Errorf("googleplay: read upload response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseGoogleAPIError(resp.StatusCode, respBody)
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("googleplay: decode upload response: %w", err)
		}
	}
	return nil
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, query url.Values, payload any, out any) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		raw, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return fmt.Errorf("googleplay: marshal request: %w", marshalErr)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpointURL(endpoint, query), body)
	if err != nil {
		return fmt.Errorf("googleplay: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("googleplay: %s %s failed: %w", method, endpoint, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return fmt.Errorf("googleplay: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseGoogleAPIError(resp.StatusCode, respBody)
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("googleplay: decode response: %w", err)
	}

	return nil
}

func (c *Client) token(ctx context.Context) (string, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(c.account.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("googleplay auth: invalid service account private key: %w", err)
	}

	now := c.now().UTC()
	claims := jwt.MapClaims{
		"iss":   c.account.ClientEmail,
		"scope": androidPublisherScope,
		"aud":   c.tokenURL,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}

	signedJWT := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if kid := strings.TrimSpace(c.account.PrivateKeyID); kid != "" {
		signedJWT.Header["kid"] = kid
	}

	assertion, err := signedJWT.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("googleplay auth: sign JWT assertion: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.tokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("googleplay auth: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("googleplay auth: request access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return "", fmt.Errorf("googleplay auth: read token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", parseOAuthError(resp.StatusCode, body)
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("googleplay auth: decode token response: %w", err)
	}
	if strings.TrimSpace(tokenResponse.AccessToken) == "" {
		return "", fmt.Errorf("googleplay auth: token response missing access_token")
	}
	return tokenResponse.AccessToken, nil
}

func (c *Client) endpointURL(endpoint string, query url.Values) string {
	base := strings.TrimRight(c.baseURL, "/")
	path := "/" + strings.TrimLeft(endpoint, "/")
	full := base + path
	if len(query) > 0 {
		return full + "?" + query.Encode()
	}
	return full
}

func parseOAuthError(statusCode int, body []byte) error {
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &payload)

	detail := strings.TrimSpace(payload.ErrorDescription)
	if detail == "" {
		detail = strings.TrimSpace(payload.Error)
	}
	if detail == "" {
		detail = strings.TrimSpace(string(body))
	}
	if detail == "" {
		detail = http.StatusText(statusCode)
	}

	return &APIError{
		Code:       oauthStatusCode(statusCode),
		Title:      "Google OAuth error",
		Detail:     detail,
		StatusCode: statusCode,
	}
}

func parseGoogleAPIError(statusCode int, body []byte) error {
	var payload struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)

	detail := strings.TrimSpace(payload.Error.Message)
	if detail == "" {
		detail = strings.TrimSpace(string(body))
	}
	if detail == "" {
		detail = http.StatusText(statusCode)
	}

	code := normalizeGoogleStatusCode(statusCode, payload.Error.Status)
	return &APIError{
		Code:       code,
		Title:      "Google Play API error",
		Detail:     detail,
		StatusCode: statusCode,
	}
}

func oauthStatusCode(statusCode int) string {
	switch statusCode {
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusBadRequest, http.StatusUnauthorized:
		return "UNAUTHORIZED"
	default:
		return statusCodeToCode(statusCode)
	}
}

func normalizeGoogleStatusCode(statusCode int, googleStatus string) string {
	normalized := strings.ToUpper(strings.TrimSpace(googleStatus))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")

	switch normalized {
	case "UNAUTHENTICATED":
		return "UNAUTHORIZED"
	case "PERMISSION_DENIED":
		return "FORBIDDEN"
	case "NOT_FOUND":
		return "NOT_FOUND"
	case "ALREADY_EXISTS":
		return "CONFLICT"
	case "INVALID_ARGUMENT":
		return "BAD_REQUEST"
	}

	if normalized != "" {
		return normalized
	}
	return statusCodeToCode(statusCode)
}

func statusCodeToCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	default:
		return ""
	}
}

func normalizeDeveloperParent(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("developer is required")
	}
	if !strings.HasPrefix(value, "developers/") {
		value = "developers/" + value
	}
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] != "developers" || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("developer must be a developer ID or resource name developers/{developer}")
	}
	if strings.ContainsAny(parts[1], " /") {
		return "", fmt.Errorf("developer must be a developer ID or resource name developers/{developer}")
	}
	return value, nil
}

func resolveRequestTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv(timeoutEnvVar)); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	if raw := strings.TrimSpace(os.Getenv(timeoutSecondsEnvVar)); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultRequestTimeout
}
