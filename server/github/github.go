package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"

	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

var DefaultTemplate = template.Must(template.New("image").Parse(`{{ .Repository.FullName }}:{{ .Deployment.Sha }}`))

type Options struct {
	// The GitHub secret to ensure that the request was sent from GitHub.
	Secret string

	// If provided, specifies the environments that this Empire instance
	// should handle deployments for.
	Environments []string

	Deployer Deployer
}

func New(e *empire.Empire, opts Options) httpx.Handler {
	r := hookshot.NewRouter()

	secret := opts.Secret
	r.Handle("deployment", hookshot.Authorize(&DeploymentHandler{Deployer: opts.Deployer, environments: opts.Environments}, secret))
	r.Handle("ping", hookshot.Authorize(http.HandlerFunc(Ping), secret))

	return r
}

// Deployment is an http.Handler for handling the `deployment` event.
type DeploymentHandler struct {
	Deployer
	environments []string
}

func (h *DeploymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("expected ServeHTTPContext to be called")
}

func (h *DeploymentHandler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var p events.Deployment

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	if !currentEnvironment(p.Deployment.Environment, h.environments) {
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "Ignore deployment to environment: %s", p.Deployment.Environment)
		return nil
	}
	if err := h.Deploy(ctx, p, os.Stdout); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, "Ok\n")
	return nil
}

func currentEnvironment(eventEnv string, environments []string) bool {
	for _, env := range environments {
		if env == eventEnv {
			return true
		}
	}
	return false
}

func Ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ok\n")
}
