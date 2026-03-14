package googleplay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClientCreateEdit(t *testing.T) {
	tokenCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error: %v", err)
			}
			if got := r.Form.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
				t.Fatalf("grant_type = %q, want jwt-bearer", got)
			}
			if strings.TrimSpace(r.Form.Get("assertion")) == "" {
				t.Fatal("expected non-empty assertion")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/androidpublisher/v3/applications/com.example.app/edits":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
				t.Fatalf("authorization header = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                "edit-1",
				"expiryTimeSeconds": "1700000000",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	edit, err := client.CreateEdit(context.Background(), "com.example.app")
	if err != nil {
		t.Fatalf("CreateEdit() error: %v", err)
	}
	if edit.ID != "edit-1" {
		t.Fatalf("edit ID = %q, want %q", edit.ID, "edit-1")
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
}

func TestClientListTracksMapsNotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":{"code":404,"message":"app not found","status":"NOT_FOUND"}}`)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.ListTracks(context.Background(), "com.example.app", "edit-1")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
	if apiErr.Code != "NOT_FOUND" {
		t.Fatalf("code = %q, want NOT_FOUND", apiErr.Code)
	}
}

func TestClientUploadBundle(t *testing.T) {
	tmpDir := t.TempDir()
	bundlePath := filepath.Join(tmpDir, "app-release.aab")
	if err := os.WriteFile(bundlePath, []byte("bundle-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	tokenCalls := 0
	uploadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/upload/androidpublisher/v3/applications/com.example.app/edits/edit-1/bundles":
			uploadCalls++
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if got := r.URL.Query().Get("uploadType"); got != "media" {
				t.Fatalf("uploadType = %q, want media", got)
			}
			if got := r.Header.Get("Content-Type"); got != "application/octet-stream" {
				t.Fatalf("content-type = %q, want application/octet-stream", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error: %v", err)
			}
			if string(body) != "bundle-bytes" {
				t.Fatalf("upload body = %q, want %q", string(body), "bundle-bytes")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"versionCode": "456",
				"sha1":        "abc",
				"sha256":      "def",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	bundle, err := client.UploadBundle(context.Background(), "com.example.app", "edit-1", bundlePath)
	if err != nil {
		t.Fatalf("UploadBundle() error: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
	if uploadCalls != 1 {
		t.Fatalf("upload calls = %d, want 1", uploadCalls)
	}
	if bundle.VersionCode.Int64() != 456 {
		t.Fatalf("version code = %d, want 456", bundle.VersionCode.Int64())
	}
}

func TestClientUploadAPK(t *testing.T) {
	tmpDir := t.TempDir()
	apkPath := filepath.Join(tmpDir, "app-release.apk")
	if err := os.WriteFile(apkPath, []byte("apk-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	tokenCalls := 0
	uploadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/upload/androidpublisher/v3/applications/com.example.app/edits/edit-1/apks":
			uploadCalls++
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if got := r.URL.Query().Get("uploadType"); got != "media" {
				t.Fatalf("uploadType = %q, want media", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error: %v", err)
			}
			if string(body) != "apk-bytes" {
				t.Fatalf("upload body = %q, want %q", string(body), "apk-bytes")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"versionCode": 789,
				"sha1":        "apksha1",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	apk, err := client.UploadAPK(context.Background(), "com.example.app", "edit-1", apkPath)
	if err != nil {
		t.Fatalf("UploadAPK() error: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
	if uploadCalls != 1 {
		t.Fatalf("upload calls = %d, want 1", uploadCalls)
	}
	if apk.VersionCode.Int64() != 789 {
		t.Fatalf("version code = %d, want 789", apk.VersionCode.Int64())
	}
}

func TestClientGetTrack(t *testing.T) {
	tokenCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/androidpublisher/v3/applications/com.example.app/edits/edit-1/tracks/production":
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"track": "production",
				"releases": []map[string]any{
					{
						"status":       "completed",
						"versionCodes": []any{"123", 124},
						"releaseNotes": []map[string]any{
							{"language": "en-US", "text": "Bug fixes"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	track, err := client.GetTrack(context.Background(), "com.example.app", "edit-1", "production")
	if err != nil {
		t.Fatalf("GetTrack() error: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
	if track.Track != "production" {
		t.Fatalf("track = %q, want production", track.Track)
	}
	if len(track.Releases) != 1 || len(track.Releases[0].ReleaseNotes) != 1 {
		t.Fatalf("unexpected release notes payload: %#v", track.Releases)
	}
	if len(track.Releases[0].VersionCodes) != 2 || track.Releases[0].VersionCodes[0] != 123 || track.Releases[0].VersionCodes[1] != 124 {
		t.Fatalf("unexpected version codes: %#v", track.Releases[0].VersionCodes)
	}
}

func TestClientListUsers(t *testing.T) {
	tokenCalls := 0
	userCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/androidpublisher/v3/developers/123456789/users":
			userCalls++
			if got := r.URL.Query().Get("pageSize"); got != "-1" {
				t.Fatalf("pageSize = %q, want -1", got)
			}
			switch pageToken := r.URL.Query().Get("pageToken"); pageToken {
			case "":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"users": []map[string]any{
						{
							"email": "owner@example.com",
							"grants": []map[string]any{
								{"packageName": "com.example.alpha"},
							},
						},
					},
					"nextPageToken": "next-token",
				})
			case "next-token":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"users": []map[string]any{
						{
							"email": "qa@example.com",
							"grants": []map[string]any{
								{"packageName": "com.example.beta"},
							},
						},
					},
				})
			default:
				t.Fatalf("unexpected pageToken = %q", pageToken)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	page1, err := client.ListUsers(context.Background(), "123456789", -1, "")
	if err != nil {
		t.Fatalf("ListUsers(page1) error: %v", err)
	}
	if page1.NextPageToken != "next-token" {
		t.Fatalf("nextPageToken = %q, want next-token", page1.NextPageToken)
	}
	if len(page1.Users) != 1 || page1.Users[0].Email != "owner@example.com" {
		t.Fatalf("unexpected page1 users: %#v", page1.Users)
	}

	page2, err := client.ListUsers(context.Background(), "developers/123456789", -1, page1.NextPageToken)
	if err != nil {
		t.Fatalf("ListUsers(page2) error: %v", err)
	}
	if len(page2.Users) != 1 || page2.Users[0].Email != "qa@example.com" {
		t.Fatalf("unexpected page2 users: %#v", page2.Users)
	}
	if tokenCalls != 2 {
		t.Fatalf("token calls = %d, want 2", tokenCalls)
	}
	if userCalls != 2 {
		t.Fatalf("user calls = %d, want 2", userCalls)
	}
}

func TestClientListUsersRejectsInvalidDeveloper(t *testing.T) {
	client, err := NewClient(testServiceAccount(t))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.ListUsers(context.Background(), "", -1, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "developer is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientListListings(t *testing.T) {
	tokenCalls := 0
	listCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/androidpublisher/v3/applications/com.example.app/edits/edit-1/listings":
			listCalls++
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"kind": "androidpublisher#listingsListResponse",
				"listings": []map[string]any{
					{
						"language":         "en-US",
						"title":            "Cat Breed ID",
						"shortDescription": "Identify cat breeds",
						"fullDescription":  "A complete description",
					},
					{
						"language": "de-DE",
						"title":    "Katzenrassen ID",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	resp, err := client.ListListings(context.Background(), "com.example.app", "edit-1")
	if err != nil {
		t.Fatalf("ListListings() error: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
	if listCalls != 1 {
		t.Fatalf("list calls = %d, want 1", listCalls)
	}
	if len(resp.Listings) != 2 {
		t.Fatalf("listings len = %d, want 2", len(resp.Listings))
	}
	if resp.Listings[0].Language != "en-US" {
		t.Fatalf("listing language = %q, want en-US", resp.Listings[0].Language)
	}
}

func TestClientGetListing(t *testing.T) {
	tokenCalls := 0
	getCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/androidpublisher/v3/applications/com.example.app/edits/edit-1/listings/en-US":
			getCalls++
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"language":         "en-US",
				"title":            "Cat Breed ID",
				"shortDescription": "Identify cat breeds",
				"fullDescription":  "A complete description",
				"video":            "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	resp, err := client.GetListing(context.Background(), "com.example.app", "edit-1", "en-US")
	if err != nil {
		t.Fatalf("GetListing() error: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
	if getCalls != 1 {
		t.Fatalf("get calls = %d, want 1", getCalls)
	}
	if resp.Title != "Cat Breed ID" {
		t.Fatalf("title = %q, want Cat Breed ID", resp.Title)
	}
}

func TestClientPatchListing(t *testing.T) {
	tokenCalls := 0
	patchCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token-123",
			})
		case "/androidpublisher/v3/applications/com.example.app/edits/edit-1/listings/en-US":
			patchCalls++
			if r.Method != http.MethodPatch {
				t.Fatalf("method = %s, want PATCH", r.Method)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error: %v", err)
			}
			if got := payload["title"]; got != "New title" {
				t.Fatalf("payload title = %#v, want %q", got, "New title")
			}
			if got := payload["shortDescription"]; got != "New short" {
				t.Fatalf("payload shortDescription = %#v, want %q", got, "New short")
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"language":         "en-US",
				"title":            "New title",
				"shortDescription": "New short",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		testServiceAccount(t),
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithTokenURL(server.URL+"/token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	updated, err := client.PatchListing(
		context.Background(),
		"com.example.app",
		"edit-1",
		"en-US",
		Listing{
			Title:            "New title",
			ShortDescription: "New short",
		},
	)
	if err != nil {
		t.Fatalf("PatchListing() error: %v", err)
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
	if patchCalls != 1 {
		t.Fatalf("patch calls = %d, want 1", patchCalls)
	}
	if updated.Language != "en-US" {
		t.Fatalf("language = %q, want en-US", updated.Language)
	}
}

func testServiceAccount(t *testing.T) ServiceAccount {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if pemBytes == nil {
		t.Fatal("failed to encode private key")
	}

	return ServiceAccount{
		Type:         "service_account",
		ProjectID:    "demo",
		PrivateKeyID: "kid",
		PrivateKey:   string(pemBytes),
		ClientEmail:  "svc@example.iam.gserviceaccount.com",
		TokenURI:     "https://oauth2.googleapis.com/token",
	}
}
