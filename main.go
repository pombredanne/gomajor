package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/icholy/gomajor/importpaths"
	"github.com/icholy/gomajor/latest"
	"github.com/icholy/gomajor/packages"
)

func main() {
	var rewrite, goget bool
	flag.BoolVar(&rewrite, "rewrite", true, "rewrite import paths")
	flag.BoolVar(&goget, "get", true, "run go get")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("missing package spec")
	}
	// figure out the correct import path
	pkgpath, version := packages.SplitSpec(flag.Arg(0))
	pkg, err := packages.Load(pkgpath)
	if err != nil {
		log.Fatal(err)
	}
	// figure out what version to get
	if version == "latest" {
		version, err = latest.Version(pkg.ModPrefix)
		if err != nil {
			log.Fatal(err)
		}
	}
	if version != "" && !semver.IsValid(version) {
		log.Fatalf("invalid version: %s", version)
	}
	// go get
	if goget {
		spec := pkg.Path(version)
		if version != "" {
			spec += "@" + semver.Canonical(version)
		}
		fmt.Println("go get", spec)
		cmd := exec.Command("go", "get", spec)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
	// rewrite imports
	if !rewrite {
		return
	}
	err = importpaths.Rewrite(".", func(name, path string) (string, bool) {
		if strings.Contains(name, "vendor"+string(filepath.Separator)) {
			return "", false
		}
		modpath, ok := pkg.FindModPath(path)
		if !ok {
			return "", false
		}
		pkgdir := strings.TrimPrefix(path, modpath)
		pkgdir = strings.TrimPrefix(pkgdir, "/")
		if pkg.PkgDir != "" && pkg.PkgDir != pkgdir {
			return "", false
		}
		newpath := packages.Package{
			PkgDir:    pkgdir,
			ModPrefix: pkg.ModPrefix,
		}.Path(version)
		if newpath == path {
			return "", false
		}
		fmt.Printf("%s: %s -> %s\n", name, path, newpath)
		return newpath, true
	})
	if err != nil {
		log.Fatal(err)
	}
}
