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
	"net/http"

	"github.com/leg100/otf/internal/tfeapi"
	"github.com/leg100/otf/internal/utils"
)

type loginDiscovery struct {
	// Server's authorization endpoint. If given as a relative URL, it is resolved
	// from the location of the service discovery document.
	Authz string `json:"authz"`
	// The server's token endpoint. If given as a relative URL, it is resolved from
	// the location of the service discovery document.
	Token string `json:"token"`
	// The client_id value to use when making requests, as defined in RFC 6749 section 2.2.
	Client string `json:"client"`
	//  A two-element JSON array giving an inclusive range of TCP ports that
	// Terraform may use for the temporary HTTP server it will start to provide
	// the redirection endpoint for the first step of an authorization code grant.
	// Terraform opens a TCP listen port on the loopback interface in order to
	// receive the response from the server's authorization endpoint.
	Ports []int `json:"ports"`
}

var discoveryPayload = utils.MustJSONMarshal(struct {
	LoginV1   loginDiscovery `json:"login.v1"`
	ModulesV1 string         `json:"modules.v1"`
	MotdV1    string         `json:"motd.v1"`
	StateV2   string         `json:"state.v2"`
	TfeV2     string         `json:"tfe.v2"`
	TfeV21    string         `json:"tfe.v2.1"`
	TfeV22    string         `json:"tfe.v2.2"`
}{
	LoginV1: loginDiscovery{
		Authz:  AuthRoute,
		Token:  TokenRoute,
		Client: OAuthClientID,
		Ports:  []int{10000, 10010},
	},
	ModulesV1: tfeapi.ModuleV1Prefix,
	MotdV1:    "/api/terraform/motd",
	StateV2:   tfeapi.APIPrefixV2,
	TfeV2:     tfeapi.APIPrefixV2,
	TfeV21:    tfeapi.APIPrefixV2,
	TfeV22:    tfeapi.APIPrefixV2,
})

func (s *TerraformAPIService) Discovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(discoveryPayload)
}
