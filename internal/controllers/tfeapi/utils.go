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

package tfeapi

import (
	"io"
	"net/http"

	"github.com/DataDog/jsonapi"
)

func unmarshal(r io.Reader, v any) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return jsonapi.Unmarshal(b, v)
}

const (
	// headerSource is an http header providing the source of the API call
	headerSource = "X-Terraform-Integration"
	// headerSourceCLI is an http header value for headerSource that indicates
	// the source of the API call is the terraform CLI
	headerSourceCLI = "cloud"
)

func isTerraformCLI(r *http.Request) bool {
	return r.Header.Get(headerSource) == headerSourceCLI
}
