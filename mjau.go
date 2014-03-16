// Copyright (c) 2012, Robert Dinu. All rights reserved.
// Use of this source code is governed by a BSD-style
// license which can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	ihttp "github.com/hybrid-publishing-lab/mjau/http" // Internal http package.
	"github.com/hybrid-publishing-lab/mjau/inventory"
	"github.com/hybrid-publishing-lab/mjau/util"
	"github.com/hybrid-publishing-lab/mjau/whitelist"
	//"github.com/SlyMarbo/spdy"
)

const (
	ProgName    = "mjau"
	ProgVersion = "0.1"
)

var (
	bFlag = flag.String("b", "0.0.0.0:80", "TCP address to bind to")
	eFlag = flag.Bool("e", false, "toggle entity tags validation")
	gFlag = flag.Bool("g", false, "toggle response gzip compression")
	lFlag = flag.String("l", "fonts/", "path to font library")
	mFlag = flag.Uint64("m", 2592000, "Cache-Control max-age value")
	oFlag = flag.Bool("o", false, "toggle cross-origin resource sharing")
	tFlag = flag.String("t", "templates/", "path to templates directory")
	vFlag = flag.Bool("v", false, "display version number and exit")
	wFlag = flag.String("w", "whitelist.json", "path to whitelist file")
)

type Error string

func init() {
	util.BlankStrFlagDefault(bFlag, "b")
	util.BlankStrFlagDefault(lFlag, "l")
	util.BlankStrFlagDefault(tFlag, "t")
	util.BlankStrFlagDefault(wFlag, "w")
	*lFlag = filepath.FromSlash(*lFlag)
	*wFlag = filepath.FromSlash(*wFlag)
}

// Error reports a generic error.
func (e Error) Error() string {
	return ProgName + ": " + string(e)
}

// PrintErrorExit prints the given error message to standard error
// and exits the program signaling abnormal termination.
func PrintErrorExit(message string) {
	fmt.Fprintln(os.Stderr, Error(message))
	os.Exit(1)
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
	flag.Parse()
	if *vFlag {
		fmt.Println(ProgName, ProgVersion)
		os.Exit(0)
	}
	// Build font inventory.
	fontInventory := inventory.New()
	if err := fontInventory.Build(*lFlag); err != nil {
		PrintErrorExit(err.Error())
	}
	if fontInventory.Len() == 0 {
		PrintErrorExit(fmt.Sprintf("%s: empty font library", *lFlag))
	}
	// Read whitelist.
	whitelist := whitelist.New()
	if err := whitelist.Read(*wFlag); err != nil {
		PrintErrorExit(err.Error())
	}
	if whitelist.Size() == 0 {
		PrintErrorExit(fmt.Sprintf("%s: empty whitelist", *wFlag))
	}
	// Parse templates.
	templatesPath := filepath.FromSlash(*tFlag)
	eot := filepath.Join(templatesPath, "eot.css.tmpl")
	woff := filepath.Join(templatesPath, "woff.css.tmpl")
	ttf := filepath.Join(templatesPath, "ttf.css.tmpl")
	odt := filepath.Join(templatesPath, "odt.css.tmpl")
	var templates *template.Template
	var err error
	if templates, err = template.ParseFiles(eot, woff, ttf, odt); err != nil {
		PrintErrorExit(err.Error())
	}
	// Create CSS handler function.
	var cssHandler http.HandlerFunc
	ctx := ihttp.HandlerContext{
		Flags: ihttp.Flags{
			AcAllowOrigin: *oFlag,
			CcMaxAge:      *mFlag,
			Etag:          *eFlag,
			Gzip:          *gFlag,
			Version:       ProgName + "/" + ProgVersion,
		},
		Inventory: *fontInventory,
		Templates: *templates,
		Whitelist: *whitelist,
	}
	cssHandler = ihttp.MakeHandler(ihttp.CssHandler, ctx)
	if *gFlag {
		// Enable response gzip compression.
		cssHandler = ihttp.MakeGzipHandler(cssHandler)
	}

	// Register CSS HTTP handler.
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(*lFlag))))
	http.HandleFunc("/css/", cssHandler)
	// Start HTTP server.
	if err := http.ListenAndServeTLS(*bFlag, "cert.pem", "key.pem", nil); err != nil {
		PrintErrorExit(err.Error())
	}
}
