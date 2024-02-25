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
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/leg100/otf/internal"
	"github.com/leg100/otf/internal/configversion"
	"github.com/leg100/otf/internal/resource"
	"github.com/leg100/otf/internal/tfeapi"
	"github.com/leg100/surl"
)

type (
	TerraformEnterpriseAPIService struct {
		cv ConfigurationVersionService

		responder *tfeapi.Responder
		signer    *surl.Signer

		maxUploadSize int64
	}

	Options struct {
		ConfigurationVersionService

		*tfeapi.Responder
		*surl.Signer

		MaxUploadSize int64
	}

	ConfigurationVersionService = configversion.ConfigurationVersionService
)

func NewTerraformEnterpriseAPIService(opts Options) *TerraformEnterpriseAPIService {
	return &TerraformEnterpriseAPIService{
		cv:            opts.ConfigurationVersionService,
		responder:     opts.Responder,
		signer:        opts.Signer,
		maxUploadSize: opts.MaxUploadSize,
	}
}

const (
	// APIPrefixV2 is the URL path prefix for TFE API endpoints
	APIPrefixV2 = "/api/v2"
)

func (s *TerraformEnterpriseAPIService) AddHandlers(r *mux.Router) {
	signed := r.PathPrefix("/signed/{signature.expiry}").Subrouter()

	r = r.PathPrefix(APIPrefixV2).Subrouter()
	r.Use(addTFEApiVersionHeaderHandler)

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	rsp := s.responder

	// Configuration Versions
	r.HandleFunc("/configuration-versions/{id}",
		h(rsp, s.getConfigurationVersion)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}/configuration-versions",
		hp(rsp, s.listConfigurationVersions)).Methods("GET")
	r.HandleFunc("/workspaces/{workspace_id}/configuration-versions",
		hc(rsp, s.createConfigurationVersion, http.StatusCreated)).Methods("POST")
	r.HandleFunc("/configuration-versions/{id}/download", s.downloadConfigurationVersion).Methods("GET")
	// Upload is *not* rooted at /api/v2
	signed.Use(internal.VerifySignedURL(s.signer))
	signed.HandleFunc("/configuration-versions/{id}/upload", s.UploadConfigurationVersion).Methods("PUT")
	rsp.Register(tfeapi.IncludeConfig, s.includeByConfigurationVersionIDField)
	rsp.Register(tfeapi.IncludeIngress, s.includeByConfigurationVersionIngressAttributes)
}

func addTFEApiVersionHeaderHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add TFP API version header to every API response.
		//
		// Version 2.5 is the minimum version terraform requires for the
		// newer 'cloud' configuration block:
		// https://developer.hashicorp.com/terraform/cli/cloud/settings#the-cloud-block
		w.Header().Set("TFP-API-Version", "2.5")

		// Remove trailing slash from all requests
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")

		next.ServeHTTP(w, r)
	})
}

func h[T any](rsp *tfeapi.Responder, m func(*http.Request) (*T, error)) http.HandlerFunc {
	return hc[T](rsp, m, http.StatusOK)
}

func hc[T any](rsp *tfeapi.Responder, m func(*http.Request) (*T, error), code int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, err := m(r)
		if err != nil {
			tfeapi.Error(w, err)
			return
		}
		rsp.Respond(w, r, res, code)
	})
}

func hp[T any](rsp *tfeapi.Responder, m func(*http.Request) ([]*T, *resource.Pagination, error)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, p, err := m(r)
		if err != nil {
			tfeapi.Error(w, err)
			return
		}
		rsp.RespondWithPage(w, r, res, p)
	})
}
