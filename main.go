package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/codegangsta/cli"
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
	ImportPrefix     string
	VCS              string
	RepoRoot         string
	RedirectURL      string
	ListenAddress    string
	TLSListenAddress string
	TLSCertFile      string
	TLSKeyFile       string
}

type context struct {
	config
	Package string
}

var (
	html   *template.Template
	cfg    config
	app    = cli.NewApp()
	engine = gin.Default()
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
			Name:   "vcs",
			EnvVar: "VCS",
			Value:  "git",
			Usage:  "vcs repo type",
		},
		cli.StringFlag{
			Name:   "repo-root",
			EnvVar: "REPO_ROOT",
			Usage:  "base url used for the repo package path, /<package> is appended ",
		},
		cli.StringFlag{
			Name:   "redirect-url",
			EnvVar: "REDIRECT_URL",
			Usage:  "url to redirect browsers to, if empty, redirects to repo-root/package",
		},
		cli.StringFlag{
			Name:   "listen-address",
			EnvVar: "LISTEN_ADDRESS",
			Value:  "[::]:80",
			Usage:  "address (ip/hostname and port) that the server should listen on",
		},
		cli.StringFlag{
			Name:   "tls-listen-address",
			EnvVar: "TLS_LISTEN_ADDRESS",
			Value:  "[::]:443",
			Usage:  "address (ip/hostname and port) that the server should listen on for tls requests",
		},
		cli.StringFlag{
			Name:   "tls-cert-file",
			EnvVar: "TLS_CERT_FILE",
			Usage:  "tls certificate bundle",
		},
		cli.StringFlag{
			Name:   "tls-key-file",
			EnvVar: "TLS_KEY_FILE",
			Usage:  "tls key file",
		},
	}

	engine.SetHTMLTemplate(html)
	engine.GET("/*package", handler)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatalln("app returned error", err)
	}
}

func setup(c *cli.Context) error {
	cfg = config{
		ImportPrefix:     c.String("import-prefix"),
		VCS:              c.String("vcs"),
		RepoRoot:         c.String("repo-root"),
		RedirectURL:      c.String("redirect-url"),
		ListenAddress:    c.String("listen-address"),
		TLSListenAddress: c.String("tls-listen-address"),
		TLSCertFile:      c.String("tls-cert-file"),
		TLSKeyFile:       c.String("tls-key-file"),
	}

	return nil
}

func run(c *cli.Context) {
	httpServer := &graceful.Server{
		Timeout: 10 * time.Second,
		Server:  &http.Server{Addr: cfg.ListenAddress, Handler: engine},
	}

	tlsServer := &graceful.Server{
		Timeout: 10 * time.Second,
		Server:  &http.Server{Addr: cfg.TLSListenAddress, Handler: engine},
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		// This always returns an error on exit
		log.Printf("listening for http at %s", cfg.ListenAddress)
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalln("http server failure", err)
		}
	}()

	if len(cfg.TLSCertFile) > 0 && len(cfg.TLSKeyFile) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("listening for tls at %s (%s, %s)", cfg.TLSListenAddress, cfg.TLSCertFile, cfg.TLSKeyFile)
			if err := tlsServer.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
				log.Fatalln("tls server failure", err)
			}
		}()
	}

	// wait for the graceful servers to stop
	wg.Wait()
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
