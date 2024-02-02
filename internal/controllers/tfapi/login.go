// Copyright (C) 2024 Francois Saint-Jacques
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package tfapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/leg100/otf/internal"
	"github.com/leg100/otf/internal/http/decode"
	"github.com/leg100/otf/internal/http/html"
	"github.com/leg100/otf/internal/user"
)

const (
	OAuthClientID = "terraform"

	// https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2.1
	ErrInvalidRequest          string = "invalid_request"
	ErrInvalidGrant            string = "invalid_grant"
	ErrInvalidClient           string = "invalid_client"
	ErrUnsupportedGrantType    string = "unsupported_grant_type"
	ErrUnsupportedResponseType string = "unsupported_response_type"
	ErrAccessDenied            string = "access_denied"
	ErrServerError             string = "server_error"
)

type (
	authcode struct {
		CodeChallenge       string `json:"code_challenge"`
		CodeChallengeMethod string `json:"code_challenge_method"`
		Username            string `json:"username"`
	}
)

func (s *TerraformAPIService) Auth(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientID            string `schema:"client_id"`
		CodeChallenge       string `schema:"code_challenge"`
		CodeChallengeMethod string `schema:"code_challenge_method"`
		RedirectURI         string `schema:"redirect_uri"`
		ResponseType        string `schema:"response_type"`
		State               string `schema:"state"`

		Consented bool `schema:"consented"`
	}
	if err := decode.All(&params, r); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	redirect, err := url.Parse(params.RedirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	if params.ClientID != OAuthClientID {
		http.Error(w, ErrInvalidClient, http.StatusBadRequest)
		return
	}

	// errors from hereon in are sent to the redirect URI as per RFC6749.
	tr := tokenRedirector{w: w, r: r, redirect: redirect, state: params.State}

	if params.ResponseType != "code" {
		tr.Error(ErrUnsupportedResponseType, "unsupported response type")
		return
	}

	if params.CodeChallengeMethod != "S256" {
		tr.Error(ErrInvalidRequest, "unsupported code challenge method")
		return
	}

	if r.Method == "GET" {
		s.renderer.Render("consent.tmpl", w, html.NewSitePage(r, "consent"))
		return
	}

	if !params.Consented {
		tr.Error(ErrAccessDenied, "user denied consent")
		return
	}

	user, err := user.UserFromContext(r.Context())
	if err != nil {
		tr.Error(ErrServerError, err.Error())
		return
	}

	marshaled, err := json.Marshal(&authcode{
		CodeChallenge:       params.CodeChallenge,
		CodeChallengeMethod: params.CodeChallengeMethod,
		Username:            user.Username,
	})
	if err != nil {
		tr.Error(ErrServerError, err.Error())
		return
	}

	encrypted, err := internal.Encrypt(marshaled, s.secret)
	if err != nil {
		tr.Error(ErrServerError, err.Error())
		return
	}

	q := redirect.Query()
	q.Add("state", params.State)
	q.Add("code", encrypted)
	redirect.RawQuery = q.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (s *TerraformAPIService) Token(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ClientID     string `schema:"client_id"`
		Code         string `schema:"code"`
		CodeVerifier string `schema:"code_verifier"`
		GrantType    string `schema:"grant_type"`
		RedirectURI  string `schema:"redirect_uri"`
	}
	if err := decode.All(&params, r); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	redirect, err := url.Parse(params.RedirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	if params.ClientID != OAuthClientID {
		http.Error(w, ErrInvalidClient, http.StatusBadRequest)
		return
	}

	// errors from hereon in are sent to the redirect URI as per RFC6749.
	tr := tokenRedirector{r: r, w: w, redirect: redirect}

	if params.Code == "" {
		tr.Error(ErrInvalidRequest, "missing code")
		return
	} else if params.CodeVerifier == "" {
		tr.Error(ErrInvalidRequest, "missing code verifier")
		return
	} else if params.GrantType != "authorization_code" {
		tr.Error(ErrUnsupportedGrantType, "")
		return
	}

	decrypted, err := internal.Decrypt(params.Code, s.secret)
	if err != nil {
		tr.Error(ErrInvalidRequest, "decrypting authentication code: "+err.Error())
		return
	}

	var code authcode
	if err := json.Unmarshal(decrypted, &code); err != nil {
		tr.Error(ErrInvalidRequest, "unmarshaling authentication code: "+err.Error())
		return
	}

	// Perform PKCE authentication
	hash := sha256.Sum256([]byte(params.CodeVerifier))
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])
	if encoded != code.CodeChallenge {
		tr.Error(ErrInvalidGrant, encoded)
		return
	}

	// Create API token for user and include in response
	userCtx := internal.AddSubjectToContext(r.Context(), &user.User{Username: code.Username})
	_, token, err := s.tok.CreateToken(userCtx, user.CreateUserTokenOptions{
		Description: "terraform login",
	})
	if err != nil {
		tr.Error(ErrInvalidRequest, err.Error())
		return
	}
	marshaled, err := json.Marshal(struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}{
		AccessToken: string(token),
		TokenType:   "bearer",
	})
	if err != nil {
		tr.Error(ErrInvalidRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(marshaled)
}

type tokenRedirector struct {
	w        http.ResponseWriter
	r        *http.Request
	redirect *url.URL
	state    string
}

func (r *tokenRedirector) Error(err, description string) {
	q := r.redirect.Query()
	q.Add("error", err)
	if description != "" {
		q.Add("error_description", description)
	}
	if r.state != "" {
		q.Add("state", r.state)
	}
	r.redirect.RawQuery = q.Encode()

	http.Redirect(r.w, r.r, r.redirect.String(), http.StatusFound)
}
