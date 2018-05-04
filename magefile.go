// +build mage

package main

import (
	"context"
	"fmt"
	"go/build"

	"github.com/magefile/mage/mg"

	"zvelo.io/zmage"
)

const (
	DockerImage = "zvelo/gopkgredir"
	Dockerfile  = "./Dockerfile"
	Exe         = "./gopkgredir"
	ExeDir      = "."
)

// Default is the default mage target
var Default = Build

// Build all executables
func Build(ctx context.Context) error {
	mg.CtxDeps(ctx, Gopkgredir)
	return nil
}

// Image builds the docker image
func Image(ctx context.Context) error {
	return zmage.BuildImage(DockerImage, Dockerfile, ".")
}

// Test runs `go vet` and `go test -race` on all packages in the repository
func Test(ctx context.Context) error {
	return zmage.GoTest(ctx)
}

// CoverOnly calculates test coverage for all packages in the repository
func CoverOnly(ctx context.Context) error {
	return zmage.CoverOnly()
}

// Cover runs CoverOnly and opens the results in the browser
func Cover(ctx context.Context) error {
	return zmage.Cover()
}

// Fmt ensures that all go files are properly formatted
func Fmt(ctx context.Context) error {
	return zmage.GoFmt()
}

// Deploy pushes the docker image with the proper tag
func Deploy(ctx context.Context) error {
	mg.CtxDeps(ctx, Image)
	return zmage.PushImage(DockerImage)
}

// Dep ensures `Gopkg.lock` and `vendor/` in sync with `Gopkg.toml`
func Dep(ctx context.Context) error {
	return zmage.Dep(ctx)
}

// Gopkgredir builds the `gopkgredir` binary
func Gopkgredir(ctx context.Context) error {
	mg.CtxDeps(ctx, Dep)
	return zmage.BuildExe(build.Default, ExeDir, Exe)
}

// Install installs all the executables
func Install(ctx context.Context) error {
	mg.CtxDeps(ctx, Dep)
	return zmage.Install(build.Default, ExeDir)
}

// Lint runs `go vet` and `golint`
func Lint(ctx context.Context) error {
	return zmage.GoLint(ctx)
}

// GoVet runs `go vet`
func GoVet(ctx context.Context) error {
	return zmage.GoVet()
}

// Clean up after yourself
func Clean(ctx context.Context) error {
	return zmage.Clean(Exe)
}

// GoPackages all the non-vendor packages in the repository
func GoPackages(ctx context.Context) error {
	pkgs, err := zmage.GoPackages(build.Default)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		fmt.Println(pkg.ImportPath, pkg.Name)
	}

	return nil
}
