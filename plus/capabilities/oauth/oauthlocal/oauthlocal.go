package oauthlocal

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"plugin"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/openrport/openrport/plus/capabilities/oauth"
	"github.com/openrport/openrport/plus/validator"
	"github.com/openrport/openrport/share/logger"
)

const (
	apiPrefix         = "/api/v1"
	defaultScope      = "openid profile email"
	deviceGrant       = "urn:ietf:params:oauth:grant-type:device_code"
	stateTTL          = 10 * time.Minute
	httpTimeout       = 15 * time.Second
	githubAPIBase     = "https://api.github.com"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	microsoftGraphURL = "https://graph.microsoft.com/v1.0"
)

var supportedProviders = map[string]struct{}{
	oauth.GitHubOAuthProvider:    {},
	oauth.MicrosoftOAuthProvider: {},
	oauth.GoogleOAuthProvider:    {},
	oauth.Auth0OAuthProvider:     {},
}

type stateEntry struct {
	expiry time.Time
}

type deviceEntry struct {
	expiresAt time.Time
}

type Capability struct {
	Provider *Provider
	Config   *oauth.Config
	Logger   *logger.Logger
}

type Provider struct {
	cfg    *oauth.Config
	client *http.Client
	logger *logger.Logger

	mu      sync.Mutex
	states  map[string]stateEntry
	devices map[string]deviceEntry
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
	ErrorURI    string `json:"error_uri"`
}

type deviceAuthResponse struct {
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
	Error           string `json:"error"`
	ErrorDesc       string `json:"error_description"`
	ErrorURI        string `json:"error_uri"`
}

type mapClaims map[string]any

func (cap *Capability) GetInitFuncName() (name string) {
	return ""
}

func (cap *Capability) InitProvider(sym plugin.Symbol) {
	if cap.Provider == nil {
		cap.Provider = &Provider{
			cfg: cap.Config,
			client: &http.Client{
				Timeout: httpTimeout,
			},
			logger:  cap.Logger,
			states:  make(map[string]stateEntry),
			devices: make(map[string]deviceEntry),
		}
	}
}

func (cap *Capability) GetOAuthCapabilityEx() (capEx oauth.CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return cap.Provider
}

func (p *Provider) ValidateConfig() (err error) {
	if p.cfg == nil {
		return oauth.ErrMissingConfig
	}
	if p.cfg.Provider == "" {
		return oauth.ErrMissingProvider
	}
	if _, ok := supportedProviders[p.cfg.Provider]; !ok {
		return fmt.Errorf("oauth provider %q is %s", p.cfg.Provider, oauth.ErrProviderNotSupportedMsg)
	}
	if p.cfg.BaseAuthorizeURL == "" {
		return oauth.ErrMissingAuthorizeURL
	}
	if _, err = url.ParseRequestURI(p.cfg.BaseAuthorizeURL); err != nil {
		return oauth.ErrInvalidAuthorizeURL
	}
	if p.cfg.TokenURL == "" {
		return oauth.ErrMissingTokenURL
	}
	if _, err = url.ParseRequestURI(p.cfg.TokenURL); err != nil {
		return oauth.ErrInvalidTokenURL
	}
	if p.cfg.RedirectURI == "" {
		return oauth.ErrMissingRedirectURI
	}
	if _, err = url.ParseRequestURI(p.cfg.RedirectURI); err != nil {
		return oauth.ErrInvalidRedirectURI
	}
	if p.cfg.ClientID == "" {
		return oauth.ErrMissingClientID
	}
	if p.cfg.ClientSecret == "" {
		return oauth.ErrMissingClientSecret
	}
	if p.cfg.BaseDeviceAuthorizeURL != "" {
		if _, err = url.ParseRequestURI(p.cfg.BaseDeviceAuthorizeURL); err != nil {
			return oauth.ErrInvalidDeviceAuthorizeURL
		}
	}

	if p.cfg.PermittedUserMatch != "" {
		re, reErr := regexp.Compile(p.cfg.PermittedUserMatch)
		if reErr != nil {
			return oauth.ErrInvalidPermittedUserMatch
		}
		p.cfg.CompiledPermittedUserMatch = re
	}
	if !p.cfg.PermittedUserList && p.cfg.PermittedUserMatch == "" {
		return oauth.ErrNeedPermittedUserControl
	}
	return nil
}

func (p *Provider) GetLoginInfo() (loginInfo *oauth.LoginInfo, err error) {
	state, err := randomState(32)
	if err != nil {
		return nil, err
	}

	expiry := time.Now().Add(stateTTL)
	p.mu.Lock()
	p.states[state] = stateEntry{expiry: expiry}
	p.purgeExpiredLocked()
	p.mu.Unlock()

	q := url.Values{}
	q.Set("client_id", p.cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", p.cfg.RedirectURI)
	q.Set("response_mode", "query")
	q.Set("scope", defaultScope)
	q.Set("state", state)

	authorizeURL := appendQuery(p.cfg.BaseAuthorizeURL, q)
	loginInfo = &oauth.LoginInfo{
		LoginMsg:     "Use your OAuth provider account to sign in.",
		AuthorizeURL: authorizeURL,
		LoginURI:     apiPrefix + oauth.DefaultLoginURI,
		State:        state,
		Expiry:       expiry,
	}
	return loginInfo, nil
}

func (p *Provider) PerformAuthCodeExchange(r *http.Request) (token string, username string, err error) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		return "", "", fmt.Errorf("missing authorization code")
	}
	if state == "" {
		return "", "", fmt.Errorf("missing state")
	}

	if err = p.validateState(state); err != nil {
		return "", "", err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", p.cfg.RedirectURI)
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)

	resp, err := p.client.PostForm(p.cfg.TokenURL, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	tr, err := decodeTokenResponse(resp.Body)
	if err != nil {
		return "", "", err
	}
	if tr.Error != "" {
		return "", "", fmt.Errorf("oauth token exchange failed: %s", fallbackError(tr.ErrorDesc, tr.Error))
	}
	if tr.AccessToken == "" {
		return "", "", fmt.Errorf("oauth token exchange returned empty access token")
	}

	claims := extractClaimsFromJWT(tr.IDToken)
	username = extractUsernameFromClaims(claims)
	if username != "" {
		if err = p.ensureUserConstraints(username, claims, tr.AccessToken); err != nil {
			return "", "", err
		}
		return tr.AccessToken, username, nil
	}

	return tr.AccessToken, "", nil
}

func (p *Provider) GetPermittedUser(r *http.Request, accessToken string) (username string, err error) {
	return p.resolvePermittedUser(accessToken)
}

func (p *Provider) GetLoginInfoForDevice(r *http.Request) (loginInfo *oauth.DeviceLoginInfo, err error) {
	if p.cfg.BaseDeviceAuthorizeURL == "" {
		return nil, fmt.Errorf("device_authorize_url is not configured")
	}

	form := url.Values{}
	clientID := p.cfg.ClientID
	if p.cfg.DeviceClientID != "" {
		clientID = p.cfg.DeviceClientID
	}
	form.Set("client_id", clientID)
	form.Set("scope", defaultScope)
	if p.cfg.DeviceClientSecret != "" {
		form.Set("client_secret", p.cfg.DeviceClientSecret)
	}

	resp, err := p.client.PostForm(p.cfg.BaseDeviceAuthorizeURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dr := &deviceAuthResponse{}
	if err = json.NewDecoder(resp.Body).Decode(dr); err != nil {
		return nil, err
	}
	if dr.Error != "" {
		return nil, fmt.Errorf("device authorization request failed: %s", fallbackError(dr.ErrorDesc, dr.Error))
	}
	if dr.DeviceCode == "" {
		return nil, fmt.Errorf("device authorization request returned empty device code")
	}

	expiresAt := time.Now().Add(time.Duration(dr.ExpiresIn) * time.Second)
	p.mu.Lock()
	p.devices[dr.DeviceCode] = deviceEntry{expiresAt: expiresAt}
	p.purgeExpiredLocked()
	p.mu.Unlock()

	loginInfo = &oauth.DeviceLoginInfo{
		LoginURI: apiPrefix + oauth.DefaultDeviceLoginURI,
		DeviceAuthInfo: &oauth.DeviceAuthInfo{
			UserCode:        dr.UserCode,
			DeviceCode:      dr.DeviceCode,
			VerificationURI: dr.VerificationURI,
			ExpiresIn:       dr.ExpiresIn,
			Interval:        dr.Interval,
			Message:         dr.Message,
		},
	}
	return loginInfo, nil
}

func (p *Provider) GetAccessTokenForDevice(r *http.Request) (token string, username string, errInfo *oauth.DeviceAuthStatusErrorInfo, err error) {
	deviceCode := r.URL.Query().Get("device_code")
	if deviceCode == "" {
		return "", "", nil, fmt.Errorf("missing device_code")
	}

	p.mu.Lock()
	entry, ok := p.devices[deviceCode]
	if !ok || time.Now().After(entry.expiresAt) {
		p.mu.Unlock()
		return "", "", nil, fmt.Errorf("device session not found or expired")
	}
	p.mu.Unlock()

	form := url.Values{}
	form.Set("grant_type", deviceGrant)
	form.Set("device_code", deviceCode)
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)

	resp, err := p.client.PostForm(p.cfg.TokenURL, form)
	if err != nil {
		return "", "", nil, err
	}
	defer resp.Body.Close()

	tr, err := decodeTokenResponse(resp.Body)
	if err != nil {
		return "", "", nil, err
	}
	if tr.Error != "" {
		status := http.StatusUnauthorized
		switch tr.Error {
		case "authorization_pending":
			status = http.StatusAccepted
		case "slow_down":
			status = http.StatusTooManyRequests
		}
		return "", "", &oauth.DeviceAuthStatusErrorInfo{
			StatusCode:   status,
			ErrorCode:    tr.Error,
			ErrorMessage: tr.ErrorDesc,
			ErrorURI:     tr.ErrorURI,
		}, fmt.Errorf("device authorization status error")
	}
	if tr.AccessToken == "" {
		return "", "", nil, fmt.Errorf("device token response returned empty access token")
	}

	claims := extractClaimsFromJWT(tr.IDToken)
	username = extractUsernameFromClaims(claims)
	if username != "" {
		if err = p.ensureUserConstraints(username, claims, tr.AccessToken); err != nil {
			return "", "", nil, err
		}
	}

	return tr.AccessToken, username, nil, nil
}

func (p *Provider) GetPermittedUserForDevice(r *http.Request, accessToken string) (username string, err error) {
	return p.resolvePermittedUser(accessToken)
}

func (p *Provider) resolvePermittedUser(accessToken string) (string, error) {
	claims := extractClaimsFromJWT(accessToken)
	username := extractUsernameFromClaims(claims)
	if username != "" {
		if err := p.ensureUserConstraints(username, claims, accessToken); err != nil {
			return "", err
		}
		return username, nil
	}

	userinfoClaims, err := p.fetchUserInfoClaims(accessToken)
	if err != nil {
		return "", err
	}

	username = extractUsernameFromClaims(userinfoClaims)
	if username == "" {
		return "", fmt.Errorf("unable to derive username from provider response")
	}
	if err = p.ensureUserConstraints(username, userinfoClaims, accessToken); err != nil {
		return "", err
	}
	return username, nil
}

func (p *Provider) ensureUserConstraints(username string, claims mapClaims, accessToken string) error {
	if !p.isPermittedUser(username) {
		return fmt.Errorf("oauth user is not permitted")
	}

	if p.cfg.RequiredOrganization != "" {
		if p.cfg.Provider != oauth.GitHubOAuthProvider {
			return fmt.Errorf("required_organization is supported only for github")
		}
		ok, err := p.isGitHubUserInRequiredOrganization(accessToken)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("oauth user is not in required github organization")
		}
	}

	if p.cfg.RequiredGroupID != "" {
		ok, err := p.isUserInRequiredGroup(claims, accessToken)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("oauth user is not in required group")
		}
	}

	return nil
}

func (p *Provider) isUserInRequiredGroup(claims mapClaims, accessToken string) (bool, error) {
	if claimsContainGroupID(claims, p.cfg.RequiredGroupID) {
		return true, nil
	}

	if p.cfg.Provider == oauth.MicrosoftOAuthProvider {
		ids, err := p.fetchMicrosoftGroupIDs(accessToken)
		if err != nil {
			return false, err
		}
		for _, id := range ids {
			if id == p.cfg.RequiredGroupID {
				return true, nil
			}
		}
		return false, nil
	}

	if p.cfg.Provider == oauth.GoogleOAuthProvider {
		return false, fmt.Errorf("required_group_id for google requires group claims in token")
	}

	return false, nil
}

func (p *Provider) isGitHubUserInRequiredOrganization(accessToken string) (bool, error) {
	orgsURL := githubAPIBase + "/user/orgs"
	respBody, err := p.getJSONWithBearer(orgsURL, accessToken)
	if err != nil {
		return false, err
	}

	var orgs []map[string]any
	if err = json.Unmarshal(respBody, &orgs); err != nil {
		return false, err
	}

	for _, org := range orgs {
		if login, ok := org["login"].(string); ok && strings.EqualFold(login, p.cfg.RequiredOrganization) {
			return true, nil
		}
	}
	return false, nil
}

func (p *Provider) fetchMicrosoftGroupIDs(accessToken string) ([]string, error) {
	memberOfURL := microsoftGraphURL + "/me/memberOf?$select=id"
	respBody, err := p.getJSONWithBearer(memberOfURL, accessToken)
	if err != nil {
		return nil, err
	}

	type memberEntry struct {
		ID string `json:"id"`
	}
	type memberResp struct {
		Value []memberEntry `json:"value"`
	}

	parsed := &memberResp{}
	if err = json.Unmarshal(respBody, parsed); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(parsed.Value))
	for _, v := range parsed.Value {
		if v.ID != "" {
			ids = append(ids, v.ID)
		}
	}
	return ids, nil
}

func (p *Provider) fetchUserInfoClaims(accessToken string) (mapClaims, error) {
	userInfoURL, err := p.getUserInfoURL()
	if err != nil {
		return nil, err
	}

	respBody, err := p.getJSONWithBearer(userInfoURL, accessToken)
	if err != nil {
		return nil, err
	}

	claims := mapClaims{}
	if err = json.Unmarshal(respBody, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (p *Provider) getUserInfoURL() (string, error) {
	switch p.cfg.Provider {
	case oauth.GitHubOAuthProvider:
		return githubAPIBase + "/user", nil
	case oauth.GoogleOAuthProvider:
		return googleUserInfoURL, nil
	case oauth.MicrosoftOAuthProvider:
		return microsoftGraphURL + "/me", nil
	case oauth.Auth0OAuthProvider:
		if p.cfg.JWKSURL != "" {
			if strings.Contains(p.cfg.JWKSURL, "/.well-known/jwks.json") {
				return strings.Replace(p.cfg.JWKSURL, "/.well-known/jwks.json", "/userinfo", 1), nil
			}
		}
		baseURL, err := url.Parse(p.cfg.BaseAuthorizeURL)
		if err != nil {
			return "", err
		}
		return baseURL.Scheme + "://" + baseURL.Host + "/userinfo", nil
	default:
		return "", fmt.Errorf("oauth provider %q is %s", p.cfg.Provider, oauth.ErrProviderNotSupportedMsg)
	}
}

func (p *Provider) getJSONWithBearer(endpoint string, accessToken string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if p.cfg.Provider == oauth.GitHubOAuthProvider {
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth provider request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (p *Provider) isPermittedUser(username string) bool {
	if p.cfg.CompiledPermittedUserMatch == nil {
		return true
	}
	return p.cfg.CompiledPermittedUserMatch.MatchString(username)
}

func (p *Provider) validateState(state string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.states[state]
	if !ok {
		return fmt.Errorf("invalid oauth state")
	}
	if time.Now().After(entry.expiry) {
		delete(p.states, state)
		return fmt.Errorf("expired oauth state")
	}
	delete(p.states, state)
	return nil
}

func (p *Provider) purgeExpiredLocked() {
	now := time.Now()
	for k, v := range p.states {
		if now.After(v.expiry) {
			delete(p.states, k)
		}
	}
	for k, v := range p.devices {
		if now.After(v.expiresAt) {
			delete(p.devices, k)
		}
	}
}

func decodeTokenResponse(r io.Reader) (*tokenResponse, error) {
	res := &tokenResponse{}
	if err := json.NewDecoder(r).Decode(res); err != nil {
		return nil, err
	}
	return res, nil
}

func appendQuery(base string, values url.Values) string {
	if strings.Contains(base, "?") {
		return base + "&" + values.Encode()
	}
	return base + "?" + values.Encode()
}

func randomState(length int) (string, error) {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func extractClaimsFromJWT(token string) mapClaims {
	if token == "" {
		return nil
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	claims := mapClaims{}
	if err = json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}

func extractUsernameFromClaims(claims mapClaims) string {
	if claims == nil {
		return ""
	}
	for _, key := range []string{"preferred_username", "email", "upn", "login", "name", "sub"} {
		if value, ok := claims[key]; ok {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

func claimsContainGroupID(claims mapClaims, requiredGroupID string) bool {
	if claims == nil || requiredGroupID == "" {
		return false
	}
	rawGroups, ok := claims["groups"]
	if !ok {
		return false
	}
	groups, ok := rawGroups.([]any)
	if !ok {
		return false
	}
	for _, g := range groups {
		if s, ok := g.(string); ok && s == requiredGroupID {
			return true
		}
	}
	return false
}

func fallbackError(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}
