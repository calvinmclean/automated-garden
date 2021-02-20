package http

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

var (
	storageClient storage.Client
	logger        *logrus.Logger
	scheduler     *gocron.Scheduler
)

// Run sets up and runs the webserver. This is the main entrypoint to our webserver application
// and is called by the "server" command
func Run(port int, plantsFilename string) {
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

	// RESTy routes for Plant API actions
	r.Route("/plants", plantRouter)

	// Read Plant information from a YAML file
	var err error
	storageClient, err = storage.NewYAMLClient(plantsFilename)
	if err != nil {
		logger.Error("Unable to initialize storage client: ", err)
		os.Exit(1)
	}

	// Initialize Scheduler and schedule watering for each existing Plant
	initializeScheduler()

	http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}

// staticHandler routes to the `./static` directory for serving static HTML and JavaScript
func staticHandler(w http.ResponseWriter, r *http.Request) {
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "static"))

	rctx := chi.RouteContext(r.Context())
	pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
	fs := http.StripPrefix(pathPrefix, http.FileServer(filesDir))
	fs.ServeHTTP(w, r)
}
