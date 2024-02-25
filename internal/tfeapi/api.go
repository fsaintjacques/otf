// Package tfeapi provides common functionality useful for implementation of the
// Hashicorp TFE/TFC API, which uses the json:api encoding
package tfeapi

import (
	"io"

	"github.com/DataDog/jsonapi"
)

const (
	// APIPrefixV2 is the URL path prefix for TFE API endpoints
	APIPrefixV2 = "/api/v2/"
	// ModuleV1Prefix is the URL path prefix for module registry endpoints
	ModuleV1Prefix = "/v1/modules/"
)

func Unmarshal(r io.Reader, v any) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return jsonapi.Unmarshal(b, v)
}
