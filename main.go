package main

import (
	"html/template"
	"net/http"
	"os"
	"path"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/tylerb/graceful"
)

const tpl = `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.ImportPrefix}}{{.Package}} {{.VCS}} {{.RepoRoot}}{{.Package}}" >
<meta http-equiv="refresh" content="0; url={{.RedirectURL}}">
</head>
<body>
Nothing to see here; <a href="{{.RedirectURL}}">move along</a>.
</body>
</html>
`

const (
	version     = "0.1.0"
	name        = "gopkgredir"
	htmlTplName = "html"
)

type config struct {
	ImportPrefix  string
	VCS           string
	RepoRoot      string
	RedirectURL   string
	ListenAddress string
}

type context struct {
	config
	Package string
}

var (
	html   *template.Template
	cfg    config
	app    = cli.NewApp()
	engine = gin.New()
)

func init() {
	html = template.Must(template.New(htmlTplName).Parse(tpl))

	app.Name = name
	app.Version = version
	app.Usage = "go package redirection service"
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
	}
	app.Before = setup
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "import-prefix, i",
			EnvVar: "IMPORT_PREFIX",
			Usage:  "base url used for the vanity url, any part of the path after that given here is considered <package>",
		},
		cli.StringFlag{
			Name:   "vcs, c",
			EnvVar: "VCS",
			Value:  "git",
			Usage:  "vcs repo type",
		},
		cli.StringFlag{
			Name:   "repo-root, u",
			EnvVar: "REPO_ROOT",
			Usage:  "base url used for the repo package path, /<package> is appended ",
		},
		cli.StringFlag{
			Name:   "redirect-url, r",
			EnvVar: "REDIRECT_URL",
			Usage:  "url to redirect browsers to, if empty, redirects to repo-root/package",
		},
		cli.StringFlag{
			Name:   "listen-address, l",
			EnvVar: "LISTEN_ADDRESS",
			Value:  "[::]:80",
			Usage:  "address (ip/hostname and port) that the server should listen on",
		},
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.WithField("error", err).Fatal("app returned error")
	}
}

func setup(c *cli.Context) error {
	gin.SetMode(gin.ReleaseMode)

	engine.Use(
		gin.Recovery(),
		ginrus.Ginrus(log.StandardLogger(), time.RFC3339, true),
		gzip.Gzip(gzip.DefaultCompression),
	)

	engine.SetHTMLTemplate(html)

	engine.GET("/*package", handler)

	cfg = config{
		ImportPrefix:  c.String("import-prefix"),
		VCS:           c.String("vcs"),
		RepoRoot:      c.String("repo-root"),
		RedirectURL:   c.String("redirect-url"),
		ListenAddress: c.String("listen-address"),
	}

	return nil
}

func run(c *cli.Context) {
	log.Infof("listening at %s", cfg.ListenAddress)
	graceful.Run(cfg.ListenAddress, 10*time.Second, engine)
}

func handler(c *gin.Context) {
	ctx := context{
		config:  cfg,
		Package: c.Param("package"),
	}

	if len(cfg.RedirectURL) == 0 {
		ctx.RedirectURL = path.Join(ctx.RepoRoot, ctx.Package)
	}

	c.HTML(http.StatusOK, htmlTplName, ctx)
}
