package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
)

type proxy struct {
}

func copyHeaders(to, from http.Header) {
	for h, vv := range from {
		for _, v := range vv {
			to.Add(h, v)
		}
	}
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	connectID := uuid.New()
	fmt.Println("HTTP Connect", connectID, r.Method, r.URL)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	r.RequestURI = ""

	resp, err := client.Do(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal("handleHTTP:", err)
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response", connectID, resp.Status)

	copyHeaders(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	fmt.Println("HTTP Close", connectID)
}

func main() {
	var addr = flag.String("addr", "0.0.0.0:8080", "The addr of the application.")
	var proto = flag.String("proto", "http", "Application protocol")
	flag.Parse()

	p := &proxy{}

	log.Println("Starting proxy server on", *addr)
	server := http.Server{
		Addr:         *addr,
		Handler:      p,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	switch *proto {
	case "http":
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf(err.Error())
		}

		break
	//case "https":
	//	if err := server.ListenAndServeTLS(options.CertFile, options.KeyFile); err != nil {
	//		log.Fatalf(err.Error())
	//	}
	//
	//	break
	default:
		log.Fatal("select http or https")
	}
}
