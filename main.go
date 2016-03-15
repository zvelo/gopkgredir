package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
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

const htmlTplName = "html"

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
	html *template.Template
	cfg  config
)

func init() {
	html = template.Must(template.New(htmlTplName).Parse(tpl))

	flag.StringVar(
		&cfg.ImportPrefix,
		"import-prefix",
		getDefaultString("IMPORT_PREFIX", ""),
		"base url used for the vanity url, any part of the path after that given here is considered <package> [$IMPORT_PREFIX]",
	)

	flag.StringVar(
		&cfg.VCS,
		"vcs",
		getDefaultString("VCS", "git"),
		"vcs repo type [$VCS]",
	)

	flag.StringVar(
		&cfg.RepoRoot,
		"repo-root",
		getDefaultString("REPO_ROOT", ""),
		"base url used for the repo package path, /<package> is appended [$REPO_ROOT]",
	)

	flag.StringVar(
		&cfg.RedirectURL,
		"redirect-url",
		getDefaultString("REDIRECT_URL", ""),
		"url to redirect browsers to, if empty, redirects to repo-root/package [$REDIRECT_URL]",
	)

	flag.StringVar(
		&cfg.ListenAddress,
		"listen-address",
		getDefaultString("LISTEN_ADDRESS", "[::]:80"),
		"address (ip/hostname and port) that the server should listen on [$LISTEN_ADDRESS]",
	)

	flag.StringVar(
		&cfg.TLSListenAddress,
		"tls-listen-address",
		getDefaultString("TLS_LISTEN_ADDRESS", "[::]:443"),
		"address (ip/hostname and port) that the server should listen on for tls requests [$TLS_LISTEN_ADDRESS]",
	)

	flag.StringVar(
		&cfg.TLSCertFile,
		"tls-cert-file",
		getDefaultString("TLS_CERT_FILE", ""),
		"tls certificate bundle [$TLS_CERT_FILE]",
	)

	flag.StringVar(
		&cfg.TLSKeyFile,
		"tls-key-file",
		getDefaultString("TLS_KEY_FILE", ""),
		"tls key file [$TLS_KEY_FILE]",
	)
}

func getDefaultString(envVar, fallback string) string {
	ret := os.Getenv(envVar)
	if len(ret) == 0 {
		return fallback
	}
	return ret
}

func main() {
	flag.Parse()
	log.Fatal(serve())
}

func serve() error {
	errCh := make(chan error, 2)

	go func() {
		log.Printf("listening for http at %s", cfg.ListenAddress)
		errCh <- http.ListenAndServe(cfg.ListenAddress, handler())
	}()

	go func() {
		if len(cfg.TLSCertFile) > 0 && len(cfg.TLSKeyFile) > 0 {
			log.Printf("listening for tls at %s (%s, %s)", cfg.TLSListenAddress, cfg.TLSCertFile, cfg.TLSKeyFile)
			errCh <- http.ListenAndServeTLS(cfg.TLSListenAddress, cfg.TLSCertFile, cfg.TLSKeyFile, handler())
		}
	}()

	return <-errCh
}

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context{
			config:  cfg,
			Package: r.URL.Path,
		}

		if len(cfg.RedirectURL) == 0 {
			ctx.RedirectURL = ctx.RepoRoot + ctx.Package
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := html.ExecuteTemplate(w, htmlTplName, ctx); err != nil {
			log.Println("error executing template", err)
		}
	})
}
