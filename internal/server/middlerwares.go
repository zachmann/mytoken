package server

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/helmet/v2"
	log "github.com/sirupsen/logrus"

	loggerUtils "github.com/oidc-mytoken/server/internal/utils/logger"
)

//go:embed web/static
var _staticFS embed.FS
var staticFS fs.FS

//go:embed web/static/img/favicon.ico
var _faviconFS embed.FS
var faviconFS fs.FS

func init() {
	var err error
	staticFS, err = fs.Sub(_staticFS, "web/static")
	if err != nil {
		log.WithError(err).Fatal()
	}
	faviconFS, err = fs.Sub(_faviconFS, "web/static/img")
	if err != nil {
		log.WithError(err).Fatal()
	}
}

func addMiddlewares(s fiber.Router) {
	addRecoverMiddleware(s)
	addFaviconMiddleware(s)
	addLoggerMiddleware(s)
	addLimiterMiddleware(s)
	addHelmetMiddleware(s)
	addStaticFiles(s)
	addCompressMiddleware(s)
}

func addLoggerMiddleware(s fiber.Router) {
	s.Use(logger.New(logger.Config{
		Format:     "${time} ${ip} ${latency} - ${status} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
		Output:     loggerUtils.MustGetAccessLogger(),
	}))
}

func addLimiterMiddleware(s fiber.Router) {
	s.Use(limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.IP() == "127.0.0.1"
		},
		Max:        100,
		Expiration: 5 * time.Minute,
	}))
}

func addCompressMiddleware(s fiber.Router) {
	s.Use(compress.New())
}

func addStaticFiles(s fiber.Router) {
	s.Use("/static", filesystem.New(filesystem.Config{
		Root:   http.FS(staticFS),
		MaxAge: 3600,
	}))
}

func addFaviconMiddleware(s fiber.Router) {
	s.Use(favicon.New(favicon.Config{
		File:       "favicon.ico",
		FileSystem: http.FS(faviconFS),
	}))
}

func addRecoverMiddleware(s fiber.Router) {
	s.Use(recover.New())
}

func addHelmetMiddleware(s fiber.Router) {
	s.Use(helmet.New())
}
