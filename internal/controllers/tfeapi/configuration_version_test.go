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
	"crypto/rand"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/leg100/otf/internal/configversion"
	"github.com/stretchr/testify/assert"
)

type fakeCVSvc struct {
	*configversion.Service
}

func (f *fakeCVSvc) Upload(ctx context.Context, cvID string, config []byte) error {
	return nil
}

func TestConfigurationVersion(t *testing.T) {
	t.Run("UploadConfigurationVersion", func(t *testing.T) {
		const maxUploadSize = 100
		svc := TerraformEnterpriseAPIService{
			cv:            &fakeCVSvc{},
			maxUploadSize: maxUploadSize,
		}

		t.Run("WithSmallPayload", func(t *testing.T) {
			reader := io.LimitReader(rand.Reader, maxUploadSize)
			req := httptest.NewRequest("PUT", "/configuration-versions/cv-1/upload?id=cv-1", reader)
			w := httptest.NewRecorder()
			svc.UploadConfigurationVersion(w, req)
			assert.Equal(t, 200, w.Code)
		})

		t.Run("WithPayloadTooBig", func(t *testing.T) {
			reader := io.LimitReader(rand.Reader, maxUploadSize+1)
			req := httptest.NewRequest("PUT", "/configuration-versions/cv-1/upload?id=cv-1", reader)
			w := httptest.NewRecorder()
			svc.UploadConfigurationVersion(w, req)
			assert.Equal(t, 422, w.Code)
		})
	})
}
