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
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/leg100/otf/internal/tfeapi"
	"github.com/stretchr/testify/require"
)

func TestDiscovery(t *testing.T) {
	srv := NewTerraformAPIService(nil, nil, nil)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.Discovery(w, r)

	require.Equal(t, 200, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	payload := w.Body.Bytes()
	var res map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &res))

	require.Equal(t, AuthRoute, res["login.v1"].(map[string]interface{})["authz"])
	require.Equal(t, TokenRoute, res["login.v1"].(map[string]interface{})["token"])
	require.Equal(t, OAuthClientID, res["login.v1"].(map[string]interface{})["client"])
	require.Equal(t, []interface{}{float64(10000), float64(10010)}, res["login.v1"].(map[string]interface{})["ports"])
	require.Equal(t, tfeapi.ModuleV1Prefix, res["modules.v1"])
	require.Equal(t, "/api/terraform/motd", res["motd.v1"])
	require.Equal(t, tfeapi.APIPrefixV2, res["state.v2"])
	require.Equal(t, tfeapi.APIPrefixV2, res["tfe.v2"])
	require.Equal(t, tfeapi.APIPrefixV2, res["tfe.v2.1"])
	require.Equal(t, tfeapi.APIPrefixV2, res["tfe.v2.2"])
}
