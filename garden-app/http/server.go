package http

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api/actions"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/sirupsen/logrus"
)

// AggregateActionRequest ...
type AggregateActionRequest struct {
	*actions.AggregateAction
}

// Bind ...
func (a *AggregateActionRequest) Bind(r *http.Request) error {
	// a.AggregateAction is nil if no AggregateAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if a.AggregateAction == nil {
		return errors.New("missing required action fields")
	}

	return nil
}

var logger *logrus.Logger

// Run ...
func Run(port int) {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	// Static handler for HTML pages
	r.Get("/*", staticHandler)

	// // RESTy routes for API actions
	r.Route("/plant/{plantID}", func(r chi.Router) {
		r.Post("/", plantAction)
	})

	http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}

func plantAction(w http.ResponseWriter, r *http.Request) {
	plant := temporaryPlantsMap[chi.URLParam(r, "plantID")]

	data := &AggregateActionRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	logger.Infof("Recieved request to perform action on Plant %s\n", plant.ID)
	data.Execute(plant)
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "static"))

	rctx := chi.RouteContext(r.Context())
	pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
	fs := http.StripPrefix(pathPrefix, http.FileServer(filesDir))
	fs.ServeHTTP(w, r)
}
