package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/mikea/gdrive-webdav/gdrive"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
	"golang.org/x/time/rate"
)

var (
	addr         = flag.String("addr", ":8765", "WebDAV service address")
	clientID     = flag.String("client-id", "", "OAuth client id")
	clientSecret = flag.String("client-secret", "", "OAuth client secret")
)

func main() {
	flag.Parse()

	if *clientID == "" {
		fmt.Fprintln(os.Stderr, "--client-id is not specified. See https://developers.google.com/drive/quickstart-go for step-by-step guide.")
		os.Exit(-1)
	}

	if *clientSecret == "" {
		fmt.Fprintln(os.Stderr, "--client-secret is not specified. See https://developers.google.com/drive/quickstart-go for step-by-step guide.")
		os.Exit(-1)
	}

	handler := &webdav.Handler{
		FileSystem: gdrive.NewFS(context.Background(), *clientID, *clientSecret),
		LockSystem: gdrive.NewLS(),
	}

	/*
		http.HandleFunc("/debug/gc", gcHandler)
		http.HandleFunc("/favicon.ico", notFoundHandler)
		http.HandleFunc("/", handler.ServeHTTP)
	*/
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/gc", gcHandler)
	mux.HandleFunc("/favicon.ico", notFoundHandler)
	mux.HandleFunc("/", handler.ServeHTTP)

	log.Info("Listening on: ", *addr)

	//err := http.ListenAndServe(*addr, nil)
	err := http.ListenAndServe(*addr, limit(mux))
	if err != nil {
		log.Errorf("Error starting HTTP server: %v", err)
		os.Exit(-1)
	}
}

// All the HTTP stuff.

/* A Limiter controls how frequently events are allowed to happen.
It implements a "token bucket" of size b, initially full and refilled at rate r tokens per second.
*/
var limiter = rate.NewLimiter(10, 1000)

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter.Allow() == false {
			log.Errorf("Limit!")
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}
		log.Errorf("No limit!")

		next.ServeHTTP(w, r)
	})
}

func gcHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("GC")
	runtime.GC()
	w.WriteHeader(200)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
}
