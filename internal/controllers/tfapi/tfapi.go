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

	"github.com/gorilla/mux"
	"github.com/leg100/otf/internal/http/html"
	"github.com/leg100/otf/internal/user"
)

type (
	TerraformAPIService struct {
		secret   []byte
		tok      tokenCreator
		renderer html.Renderer
	}

	tokenCreator interface {
		CreateToken(context.Context, user.CreateUserTokenOptions) (*user.UserToken, []byte, error)
	}
)

func NewTerraformAPIService(secret []byte, tok tokenCreator, renderer html.Renderer) *TerraformAPIService {
	return &TerraformAPIService{secret: secret, tok: tok, renderer: renderer}
}

const (
	WellknownRoute = "/.well-known/terraform.json"
	AuthRoute      = "/app/oauth2/auth"
	TokenRoute     = "/oauth2/token"
)

func (s *TerraformAPIService) AddHandlers(r *mux.Router) {
	// Implements the "remote service discovery protocol"
	// See https://developer.hashicorp.com/terraform/internals/v1.3.x/remote-service-discovery
	r.HandleFunc(WellknownRoute, s.Discovery).Methods("GET")
	// Implements the "terraform login protocol"
	// See https://developer.hashicorp.com/terraform/internals/v1.3.x/login-protocol
	r.HandleFunc(AuthRoute, s.Auth).Methods("GET", "POST")
	r.HandleFunc(TokenRoute, s.Token).Methods("POST")
}
