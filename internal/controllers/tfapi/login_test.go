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
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/leg100/otf/internal"
	"github.com/leg100/otf/internal/testutils"
	"github.com/leg100/otf/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	creator struct{}
)

func (c *creator) CreateToken(ctx context.Context, opts user.CreateUserTokenOptions) (*user.UserToken, []byte, error) {
	return nil, nil, nil
}

func TestLogin(t *testing.T) {
	secret := testutils.NewSecret(t)
	srv := NewTerraformAPIService(secret, &creator{}, testutils.NewRenderer(t))

	t.Run("AuthHandler", func(t *testing.T) {
		q := "/?"
		q += "redirect_uri=https://localhost:10000"
		q += "&client_id=terraform"
		q += "&response_type=code"
		q += "&consented=true"
		q += "&code_challenge_method=S256"
		q += "&state=somethingrandom"

		r := httptest.NewRequest("POST", q, nil)
		r = r.WithContext(internal.AddSubjectToContext(r.Context(), &user.User{Username: "bobby"}))
		w := httptest.NewRecorder()
		srv.Auth(w, r)

		// check redirect URI
		require.Equal(t, 302, w.Code)
		redirect, err := w.Result().Location()
		require.NoError(t, err)
		assert.Equal(t, "localhost:10000", redirect.Host)
		assert.Equal(t, "somethingrandom", redirect.Query().Get("state"))
		// ensure we haven't receive an oauth error payload
		require.Empty(t, redirect.Query().Get("error"))

		// check contents of auth code
		encrypted := redirect.Query().Get("code")
		decrypted, err := internal.Decrypt(encrypted, secret)
		require.NoError(t, err)
		var code authcode
		err = json.Unmarshal(decrypted, &code)
		require.NoError(t, err)
		assert.Equal(t, "bobby", code.Username)

	})
	t.Run("TokenHandler", func(t *testing.T) {
		verifier := "myverifier"
		hash := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(hash[:])

		mashaled, err := json.Marshal(&authcode{
			CodeChallenge:       challenge,
			CodeChallengeMethod: "S256",
			Username:            "bobby",
		})
		require.NoError(t, err)
		code, err := internal.Encrypt(mashaled, []byte(secret))
		require.NoError(t, err)

		q := "/?"
		q += "redirect_uri=https://localhost:10000"
		q += "&client_id=terraform"
		q += "&grant_type=authorization_code"
		q += "&code=" + code
		q += "&code_verifier=" + verifier

		r := httptest.NewRequest("POST", q, nil)
		w := httptest.NewRecorder()
		srv.Token(w, r)

		require.Equal(t, 200, w.Code, w.Body.String())

		// decrypted, err := internal.Decrypt(w.Body.String(), secret)
		// require.NoError(t, err)

		var response struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

	})
}
