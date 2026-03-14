package googleplay

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ServiceAccount represents a Google Cloud service-account key JSON payload.
type ServiceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	APIBaseURL              string `json:"-"`
}

// AppEdit is the Android Publisher "AppEdit" resource.
type AppEdit struct {
	ID                string `json:"id"`
	ExpiryTimeSeconds string `json:"expiryTimeSeconds,omitempty"`
}

// TracksListResponse is the list response for edit tracks.
type TracksListResponse struct {
	Tracks []Track `json:"tracks"`
}

// ListingsListResponse is the list response for edit listings.
type ListingsListResponse struct {
	Kind     string    `json:"kind,omitempty"`
	Listings []Listing `json:"listings,omitempty"`
}

// Listing is an Android Publisher localized store listing resource.
type Listing struct {
	Language         string `json:"language,omitempty"`
	Title            string `json:"title,omitempty"`
	FullDescription  string `json:"fullDescription,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
	Video            string `json:"video,omitempty"`
}

// Track is an Android Publisher track resource.
type Track struct {
	Track    string         `json:"track"`
	Releases []TrackRelease `json:"releases,omitempty"`
}

// ListUsersResponse is the users list response for a developer account.
type ListUsersResponse struct {
	Users         []User `json:"users,omitempty"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// User is a Play Console user with developer-level and app-level grants.
type User struct {
	Name                        string   `json:"name,omitempty"`
	Email                       string   `json:"email,omitempty"`
	AccessState                 string   `json:"accessState,omitempty"`
	Partial                     bool     `json:"partial,omitempty"`
	DeveloperAccountPermissions []string `json:"developerAccountPermissions,omitempty"`
	Grants                      []Grant  `json:"grants,omitempty"`
}

// Grant is a per-app access grant nested under a user.
type Grant struct {
	Name                string   `json:"name,omitempty"`
	PackageName         string   `json:"packageName,omitempty"`
	AppLevelPermissions []string `json:"appLevelPermissions,omitempty"`
}

// TrackRelease is an Android Publisher release resource nested under a track.
type TrackRelease struct {
	Name                string          `json:"name,omitempty"`
	Status              string          `json:"status,omitempty"`
	VersionCodes        []int64         `json:"versionCodes,omitempty"`
	ReleaseNotes        []LocalizedText `json:"releaseNotes,omitempty"`
	UserFraction        *float64        `json:"userFraction,omitempty"`
	InAppUpdatePriority *int            `json:"inAppUpdatePriority,omitempty"`
}

func (r *TrackRelease) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("nil TrackRelease")
	}

	var wire struct {
		Name                string          `json:"name,omitempty"`
		Status              string          `json:"status,omitempty"`
		VersionCodes        []Int64Value    `json:"versionCodes,omitempty"`
		ReleaseNotes        []LocalizedText `json:"releaseNotes,omitempty"`
		UserFraction        *float64        `json:"userFraction,omitempty"`
		InAppUpdatePriority *int            `json:"inAppUpdatePriority,omitempty"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	versionCodes := make([]int64, len(wire.VersionCodes))
	for i, code := range wire.VersionCodes {
		versionCodes[i] = code.Int64()
	}

	*r = TrackRelease{
		Name:                wire.Name,
		Status:              wire.Status,
		VersionCodes:        versionCodes,
		ReleaseNotes:        wire.ReleaseNotes,
		UserFraction:        wire.UserFraction,
		InAppUpdatePriority: wire.InAppUpdatePriority,
	}
	return nil
}

// LocalizedText represents localized release note text.
type LocalizedText struct {
	Language string `json:"language"`
	Text     string `json:"text"`
}

// UpdateTrackRequest is the request payload for tracks update.
type UpdateTrackRequest struct {
	Releases []TrackRelease `json:"releases"`
}

// Bundle is an uploaded Android App Bundle resource.
type Bundle struct {
	VersionCode Int64Value `json:"versionCode,omitempty"`
	SHA1        string     `json:"sha1,omitempty"`
	SHA256      string     `json:"sha256,omitempty"`
}

// APK is an uploaded APK resource.
type APK struct {
	VersionCode Int64Value `json:"versionCode,omitempty"`
	SHA1        string     `json:"sha1,omitempty"`
	SHA256      string     `json:"sha256,omitempty"`
}

// Int64Value decodes int64 JSON values represented as either number or string.
type Int64Value int64

// Int64 returns the primitive int64 value.
func (v Int64Value) Int64() int64 {
	return int64(v)
}

func (v *Int64Value) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("nil Int64Value")
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*v = 0
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			*v = 0
			return nil
		}
		parsed, err := strconv.ParseInt(asString, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int64 string %q", asString)
		}
		*v = Int64Value(parsed)
		return nil
	}

	var asNumber int64
	if err := json.Unmarshal(data, &asNumber); err == nil {
		*v = Int64Value(asNumber)
		return nil
	}

	return fmt.Errorf("invalid int64 value: %s", trimmed)
}
