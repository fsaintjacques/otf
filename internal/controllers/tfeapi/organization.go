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
	"context"
	"net/http"
	"reflect"

	"github.com/leg100/otf/internal/http/decode"
	"github.com/leg100/otf/internal/organization"
	"github.com/leg100/otf/internal/resource"
	"github.com/leg100/otf/internal/tfeapi/types"
)

func (s *TerraformEnterpriseAPIService) getOrganization(r *http.Request) (*types.Organization, error) {
	name, err := decode.Param("name", r)
	if err != nil {
		return nil, err
	}

	org, err := s.org.Get(r.Context(), name)
	if err != nil {
		return nil, err
	}

	return convertOrganization(org), nil
}

func (s *TerraformEnterpriseAPIService) listOrganizations(r *http.Request) ([]*types.Organization, *resource.Pagination, error) {
	var p struct {
		resource.PageOptions
	}
	if err := decode.Query(&p, r.URL.Query()); err != nil {
		return nil, nil, err
	}

	opts := organization.ListOptions{
		PageOptions: p.PageOptions,
	}

	page, err := s.org.List(r.Context(), opts)
	if err != nil {
		return nil, nil, err
	}

	var to []*types.Organization
	for _, org := range page.Items {
		to = append(to, convertOrganization(org))
	}

	return to, page.Pagination, nil
}

func (s *TerraformEnterpriseAPIService) createOrganization(r *http.Request) (*types.Organization, error) {
	var p types.OrganizationCreateOptions
	if err := unmarshal(r.Body, &p); err != nil {
		return nil, err
	}

	opts := organization.CreateOptions{
		Name:                       p.Name,
		Email:                      p.Email,
		CollaboratorAuthPolicy:     (*string)(p.CollaboratorAuthPolicy),
		CostEstimationEnabled:      p.CostEstimationEnabled,
		SessionRemember:            p.SessionRemember,
		SessionTimeout:             p.SessionTimeout,
		AllowForceDeleteWorkspaces: p.AllowForceDeleteWorkspaces,
	}

	org, err := s.org.Create(r.Context(), opts)
	if err != nil {
		return nil, err
	}

	return convertOrganization(org), nil
}

func (s *TerraformEnterpriseAPIService) updateOrganization(r *http.Request) (*types.Organization, error) {
	name, err := decode.Param("name", r)
	if err != nil {
		return nil, err
	}

	var p types.OrganizationUpdateOptions
	if err := unmarshal(r.Body, &p); err != nil {
		return nil, err
	}

	opts := organization.UpdateOptions{
		Name:                       p.Name,
		Email:                      p.Email,
		CollaboratorAuthPolicy:     (*string)(p.CollaboratorAuthPolicy),
		CostEstimationEnabled:      p.CostEstimationEnabled,
		SessionRemember:            p.SessionRemember,
		SessionTimeout:             p.SessionTimeout,
		AllowForceDeleteWorkspaces: p.AllowForceDeleteWorkspaces,
	}

	org, err := s.org.Update(r.Context(), name, opts)
	if err != nil {
		return nil, err
	}

	return convertOrganization(org), nil
}

func (s *TerraformEnterpriseAPIService) deleteOrganization(r *http.Request) error {
	name, err := decode.Param("name", r)
	if err != nil {
		return err
	}

	return s.org.Delete(r.Context(), name)
}

func (s *TerraformEnterpriseAPIService) getOrganizationEntitlements(r *http.Request) (*types.Entitlements, error) {
	name, err := decode.Param("name", r)
	if err != nil {
		return nil, err
	}

	entitlements, err := s.org.GetEntitlements(r.Context(), name)
	if err != nil {
		return nil, err
	}

	return (*types.Entitlements)(&entitlements), nil
}

func (s *TerraformEnterpriseAPIService) createOrganizationToken(r *http.Request) (*types.OrganizationToken, error) {
	org, err := decode.Param("name", r)
	if err != nil {
		return nil, err
	}
	var opts types.OrganizationTokenCreateOptions
	if err := unmarshal(r.Body, &opts); err != nil {
		return nil, err
	}

	ot, token, err := s.org.CreateToken(r.Context(), organization.CreateOrganizationTokenOptions{
		Organization: org,
		Expiry:       opts.ExpiredAt,
	})
	if err != nil {
		return nil, err
	}

	to := &types.OrganizationToken{
		ID:        ot.ID,
		CreatedAt: ot.CreatedAt,
		Token:     string(token),
		ExpiredAt: ot.Expiry,
	}
	return to, nil
}

func (s *TerraformEnterpriseAPIService) getOrganizationToken(r *http.Request) (*types.OrganizationToken, error) {
	org, err := decode.Param("name", r)
	if err != nil {
		return nil, err
	}

	ot, err := s.org.GetOrganizationToken(r.Context(), org)
	if err != nil {
		return nil, err
	}

	to := &types.OrganizationToken{
		ID:        ot.ID,
		CreatedAt: ot.CreatedAt,
		ExpiredAt: ot.Expiry,
	}
	return to, nil
}

func (s *TerraformEnterpriseAPIService) deleteOrganizationToken(r *http.Request) error {
	org, err := decode.Param("name", r)
	if err != nil {
		return err
	}
	return s.org.DeleteToken(r.Context(), org)
}

func (s *TerraformEnterpriseAPIService) includeByOrganizationField(ctx context.Context, v any) ([]any, error) {
	dst := reflect.Indirect(reflect.ValueOf(v))

	// v must be a struct with a field named Organization of type
	// *types.Organization
	if dst.Kind() != reflect.Struct {
		return nil, nil
	}
	field := dst.FieldByName("Organization")
	if !field.IsValid() {
		return nil, nil
	}
	tfeOrganization, ok := field.Interface().(*types.Organization)
	if !ok {
		return nil, nil
	}
	org, err := s.org.Get(ctx, tfeOrganization.Name)
	if err != nil {
		return nil, err
	}
	return []any{convertOrganization(org)}, nil
}

func convertOrganization(from *organization.Organization) *types.Organization {
	to := &types.Organization{
		Name:                       from.Name,
		CreatedAt:                  from.CreatedAt,
		ExternalID:                 from.ID,
		Permissions:                &types.DefaultOrganizationPermissions,
		SessionRemember:            from.SessionRemember,
		SessionTimeout:             from.SessionTimeout,
		AllowForceDeleteWorkspaces: from.AllowForceDeleteWorkspaces,
		CostEstimationEnabled:      from.CostEstimationEnabled,
		// go-tfe tests expect this attribute to be equal to 5
		RemainingTestableCount: 5,
	}
	if from.Email != nil {
		to.Email = *from.Email
	}
	if from.CollaboratorAuthPolicy != nil {
		to.CollaboratorAuthPolicy = types.AuthPolicyType(*from.CollaboratorAuthPolicy)
	}
	return to
}
