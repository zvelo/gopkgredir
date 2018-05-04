package main // import "zvelo.io/gopkgredir"

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const tpl = `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.ImportPrefix}}/{{.RepoName}} {{.VCS}} {{.RepoRoot}}/{{.RepoName}}" >
<meta http-equiv="refresh" content="0; url={{.RedirectRoot}}/{{.RepoName}}">
</head>
<body>
Nothing to see here; <a href="{{.RedirectRoot}}/{{.RepoName}}">move along</a>.
</body>
</html>
`

const (
	htmlTplName             = "html"
	defaultListenAddress    = "[::1]:http"
	defaultTLSListenAddress = "[::1]:https"
)

type config struct {
	ImportPrefix  string
	VCS           string
	RepoRoot      string
	RedirectRoot  string
	ListenAddress string
	TLSCertFile   string
	TLSKeyFile    string
}

type context struct {
	config
	RepoName     string
	RedirectRoot string
}

var (
	version   string
	gitCommit string
	buildDate string

	html *template.Template
	cfg  config
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nAvailable Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  version\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nFlags:\n")
		flag.PrintDefaults()
	}

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
		"base url used for the repo package path, the first path part of <package> is appended [$REPO_ROOT]",
	)

	flag.StringVar(
		&cfg.RedirectRoot,
		"redirect-root",
		getDefaultString("REDIRECT_ROOT", ""),
		"url to redirect browsers to, if empty, redirects to repo-root/package [$REDIRECT_ROOT]",
	)

	flag.StringVar(
		&cfg.ListenAddress,
		"listen-address",
		getDefaultString("LISTEN_ADDRESS", ""),
		"address (ip/hostname and port) that the server should listen on (defaults to "+defaultListenAddress+" or "+defaultTLSListenAddress+" if tls certs are defined) [$LISTEN_ADDRESS]",
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

	if len(flag.Args()) > 0 && flag.Args()[0] == "version" {
		fmt.Printf("%s (commit %s; built %s; %s)\n", version, gitCommit, buildDate, runtime.Version())
		os.Exit(0)
	}

	setupListenAddress()
	log.Fatal(serve())
}

func setupListenAddress() {
	if len(cfg.ListenAddress) != 0 {
		return
	}

	if len(cfg.TLSCertFile) > 0 && len(cfg.TLSKeyFile) > 0 {
		cfg.ListenAddress = defaultTLSListenAddress
		return
	}

	cfg.ListenAddress = defaultListenAddress
}

func serve() error {
	if len(cfg.TLSCertFile) > 0 && len(cfg.TLSKeyFile) > 0 {
		log.Printf("listening for tls at %s (%s, %s)", cfg.ListenAddress, cfg.TLSCertFile, cfg.TLSKeyFile)
		return http.ListenAndServeTLS(cfg.ListenAddress, cfg.TLSCertFile, cfg.TLSKeyFile, handler())
	}

	log.Printf("WARNING: TLS has not been configured!")
	log.Printf("listening for http at %s", cfg.ListenAddress)
	return http.ListenAndServe(cfg.ListenAddress, handler())
}

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context{
			config:       cfg,
			RedirectRoot: cfg.RedirectRoot,
		}

		pkg := strings.Split(r.URL.Path, "/")
		if len(pkg) > 1 {
			ctx.RepoName = pkg[1]

			if len(cfg.RedirectRoot) == 0 {
				ctx.RedirectRoot = ctx.RepoRoot + "/" + pkg[1]
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := html.ExecuteTemplate(w, htmlTplName, ctx); err != nil {
			log.Println("error executing template", err)
		}
	})
}
