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
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/leg100/otf/internal"
	"github.com/leg100/otf/internal/configversion"
	ihttp "github.com/leg100/otf/internal/http"
	"github.com/leg100/otf/internal/http/decode"
	"github.com/leg100/otf/internal/resource"
	"github.com/leg100/otf/internal/tfeapi"
	"github.com/leg100/otf/internal/tfeapi/types"
)

func (s *TerraformEnterpriseAPIService) getConfigurationVersion(r *http.Request) (*types.ConfigurationVersion, error) {
	id, err := decode.Param("id", r)
	if err != nil {
		return nil, err
	}

	cv, err := s.cv.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}

	return convertConfigurationVersion(cv, ""), nil
}

func (s *TerraformEnterpriseAPIService) listConfigurationVersions(r *http.Request) ([]*types.ConfigurationVersion, *resource.Pagination, error) {
	var p struct {
		WorkspaceID string `schema:"workspace_id,required"`
		resource.PageOptions
	}

	if err := decode.All(&p, r); err != nil {
		return nil, nil, err
	}

	opts := configversion.ListOptions{
		PageOptions: p.PageOptions,
	}

	page, err := s.cv.List(r.Context(), p.WorkspaceID, opts)
	if err != nil {
		return nil, nil, err
	}

	items := make([]*types.ConfigurationVersion, len(page.Items))
	for i, from := range page.Items {
		items[i] = convertConfigurationVersion(from, "")
	}

	return items, page.Pagination, nil
}

func (s *TerraformEnterpriseAPIService) downloadConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	id, err := decode.Param("id", r)
	if err != nil {
		tfeapi.Error(w, err)
		return
	}

	buf, err := s.cv.Download(r.Context(), id)
	if err != nil {
		tfeapi.Error(w, err)
		return
	}

	w.Write(buf)
}

func (s *TerraformEnterpriseAPIService) createConfigurationVersion(r *http.Request) (*types.ConfigurationVersion, error) {
	workspaceID, err := decode.Param("workspace_id", r)
	if err != nil {
		return nil, err
	}
	params := types.ConfigurationVersionCreateOptions{}
	if err := unmarshal(r.Body, &params); err != nil {
		return nil, err
	}

	source := configversion.SourceAPI
	if isTerraformCLI(r) {
		source = configversion.SourceTerraform
	}

	opts := configversion.CreateOptions{
		AutoQueueRuns: params.AutoQueueRuns,
		Speculative:   params.Speculative,
		Source:        source,
	}

	cv, err := s.cv.Create(r.Context(), workspaceID, opts)
	if err != nil {
		return nil, err
	}

	// upload url is only provided in the response when *creating* configuration version:
	//
	// https://developer.hashicorp.com/terraform/cloud-docs/api-docs/configuration-versions#configuration-files-upload-url
	url, err := s.signConfigurationVersionUploadURL(r, cv.ID)
	if err != nil {
		return nil, err
	}

	return convertConfigurationVersion(cv, url), nil
}

func (s *TerraformEnterpriseAPIService) signConfigurationVersionUploadURL(r *http.Request, ID string) (string, error) {
	url, err := s.signer.Sign(fmt.Sprintf("/configuration-versions/%s/upload", ID), time.Hour)
	if err != nil {
		return "", err
	}

	return ihttp.Absolute(r, url), nil
}

func (s *TerraformEnterpriseAPIService) UploadConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	id, err := decode.Param("id", r)
	if err != nil {
		tfeapi.Error(w, err)
		return
	}

	buf, err := io.ReadAll(io.LimitReader(r.Body, s.maxUploadSize+1))
	if err != nil {
		tfeapi.Error(w, err)
		return
	} else if int64(len(buf)) > s.maxUploadSize {
		tfeapi.Error(w, &internal.HTTPError{
			Code:    422,
			Message: fmt.Sprintf("configuration version exceeds maximum size (%d bytes)", s.maxUploadSize),
		})
	}

	if err := s.cv.Upload(r.Context(), id, buf); err != nil {
		tfeapi.Error(w, err)
		return
	}
}

func (s *TerraformEnterpriseAPIService) includeByConfigurationVersionIDField(ctx context.Context, v any) ([]any, error) {
	dst := reflect.Indirect(reflect.ValueOf(v))

	// v must be a struct with a field named ConfigurationVersionID of kind string
	if dst.Kind() != reflect.Struct {
		return nil, nil
	}
	id := dst.FieldByName("ConfigurationVersionID")
	if !id.IsValid() {
		return nil, nil
	}
	if id.Kind() != reflect.String {
		return nil, nil
	}
	cv, err := s.cv.Get(ctx, id.String())
	if err != nil {
		return nil, err
	}
	return []any{convertConfigurationVersion(cv, "")}, nil
}

func (s *TerraformEnterpriseAPIService) includeByConfigurationVersionIngressAttributes(ctx context.Context, v any) ([]any, error) {
	tfeCV, ok := v.(*types.ConfigurationVersion)
	if !ok {
		return nil, nil
	}
	if tfeCV.IngressAttributes == nil {
		return nil, nil
	}
	// the tfe CV does not by default include ingress attributes, whereas the
	// otf CV *does*, so we need to fetch it.
	cv, err := s.cv.Get(ctx, tfeCV.ID)
	if err != nil {
		return nil, err
	}
	return []any{&types.IngressAttributes{
		ID:        internal.ConvertID(cv.ID, "ia"),
		CommitSHA: cv.IngressAttributes.CommitSHA,
		CommitURL: cv.IngressAttributes.CommitURL,
	}}, nil
}

func convertConfigurationVersion(from *configversion.ConfigurationVersion, url string) *types.ConfigurationVersion {
	to := &types.ConfigurationVersion{
		ID:               from.ID,
		AutoQueueRuns:    from.AutoQueueRuns,
		Speculative:      from.Speculative,
		Source:           string(from.Source),
		Status:           string(from.Status),
		StatusTimestamps: &types.CVStatusTimestamps{},
		UploadURL:        url,
	}
	if from.IngressAttributes != nil {
		to.IngressAttributes = &types.IngressAttributes{
			ID: internal.ConvertID(from.ID, "ia"),
		}
	}
	for _, ts := range from.StatusTimestamps {
		switch ts.Status {
		case configversion.ConfigurationPending:
			to.StatusTimestamps.QueuedAt = &ts.Timestamp
		case configversion.ConfigurationErrored:
			to.StatusTimestamps.FinishedAt = &ts.Timestamp
		case configversion.ConfigurationUploaded:
			to.StatusTimestamps.StartedAt = &ts.Timestamp
		}
	}
	return to
}
