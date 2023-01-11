package gitlab

import (
	"context"
	"net/http"

	"github.com/leg100/otf"
	"github.com/leg100/otf/cloud"
)

type Cloud struct{}

func (g *Cloud) NewClient(ctx context.Context, opts otf.CloudClientOptions) (cloud.Client, error) {
	return NewClient(ctx, opts)
}

func (Cloud) HandleEvent(w http.ResponseWriter, r *http.Request, opts otf.HandleEventOptions) cloud.VCSEvent {
	return nil
}
