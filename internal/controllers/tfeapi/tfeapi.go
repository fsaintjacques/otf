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
	"github.com/leg100/otf/internal/organization"
	"github.com/leg100/otf/internal/resource"
	"github.com/leg100/otf/internal/tfeapi"
	"github.com/leg100/surl"
)

type (
	TerraformEnterpriseAPIService struct {
		cv  ConfigurationVersionService
		org OrganizationService

		responder *tfeapi.Responder
		signer    *surl.Signer

		maxUploadSize int64
	}

	Options struct {
		ConfigurationVersionService
		OrganizationService

		*tfeapi.Responder
		*surl.Signer

		MaxUploadSize int64
	}

	ConfigurationVersionService = configversion.ConfigurationVersionService
	OrganizationService         = organization.OrganizationService
)

func NewTerraformEnterpriseAPIService(opts Options) *TerraformEnterpriseAPIService {
	return &TerraformEnterpriseAPIService{
		cv:  opts.ConfigurationVersionService,
		org: opts.OrganizationService,

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
	signed.Use(internal.VerifySignedURL(s.signer))

	r = r.PathPrefix(APIPrefixV2).Subrouter()
	r.Use(addTFEApiVersionHeaderHandler)

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	rsp := s.responder

	// Configuration Versions
	r.HandleFunc("/workspaces/{workspace_id}/configuration-versions", hc(rsp, s.createConfigurationVersion, http.StatusCreated)).Methods("POST")
	r.HandleFunc("/workspaces/{workspace_id}/configuration-versions", hp(rsp, s.listConfigurationVersions)).Methods("GET")
	r.HandleFunc("/configuration-versions/{id}", h(rsp, s.getConfigurationVersion)).Methods("GET")
	r.HandleFunc("/configuration-versions/{id}/download", s.downloadConfigurationVersion).Methods("GET")
	// Upload is *not* rooted at /api/v2
	signed.HandleFunc("/configuration-versions/{id}/upload", s.UploadConfigurationVersion).Methods("PUT")
	rsp.Register(tfeapi.IncludeConfig, s.includeByConfigurationVersionIDField)
	rsp.Register(tfeapi.IncludeIngress, s.includeByConfigurationVersionIngressAttributes)

	// Organizations
	r.HandleFunc("/organizations", hc(rsp, s.createOrganization, http.StatusCreated)).Methods("POST")
	r.HandleFunc("/organizations", hp(rsp, s.listOrganizations)).Methods("GET")
	r.HandleFunc("/organizations/{name}", h(rsp, s.getOrganization)).Methods("GET")
	r.HandleFunc("/organizations/{name}", h(rsp, s.updateOrganization)).Methods("PATCH")
	r.HandleFunc("/organizations/{name}", he(rsp, s.deleteOrganization)).Methods("DELETE")
	r.HandleFunc("/organizations/{name}/entitlement-set", h(rsp, s.getOrganizationEntitlements)).Methods("GET")
	r.HandleFunc("/organizations/{name}/authentication-token", h(rsp, s.createOrganizationToken)).Methods("POST")
	r.HandleFunc("/organizations/{name}/authentication-token", h(rsp, s.getOrganizationToken)).Methods("GET")
	r.HandleFunc("/organizations/{name}/authentication-token", he(rsp, s.deleteOrganizationToken)).Methods("DELETE")
	rsp.Register(tfeapi.IncludeOrganization, s.includeByOrganizationField)
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

func he(rsp *tfeapi.Responder, m func(*http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := m(r); err != nil {
			tfeapi.Error(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
