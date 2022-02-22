package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"net/http"
	"time"
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

func connectHandshake(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return nil, err
	}

	w.WriteHeader(http.StatusOK)
	return conn, nil
}

func connectHijacker(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return nil, errors.New("hijacking not supported")
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	return conn, nil
}

func transfer(dest io.WriteCloser, src io.ReadCloser) {
	defer dest.Close()
	defer src.Close()

	io.Copy(dest, src)
}

func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	connectID := uuid.New()
	fmt.Println("HTTPS Connect", connectID, r.Method, r.URL)

	destConn, err := connectHandshake(w, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("connectHandshake:", err)
		return
	}

	fmt.Println("HTTPS handshake:", connectID)

	srcConn, err := connectHijacker(w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("connectHijacker:", err)
		return
	}

	fmt.Println("HTTPS hijacker", connectID)

	go transfer(destConn, srcConn)
	go transfer(srcConn, destConn)

	fmt.Println("HTTPS open transfer", connectID)
}

func handleHTTP(w http.ResponseWriter, r *http.Request) {
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

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		handleHTTPS(w, r)
	} else {
		handleHTTP(w, r)
	}
}

func main() {
	var addr = flag.String("addr", "0.0.0.0:8080", "The addr of the application.")
	var proto = flag.String("proto", "http", "Application protocol")
	var cert = flag.String("cert", "./certs/root.pem", "Cert file location")
	var key = flag.String("key", "./certs/root.key", "Key file location")
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
	case "https":
		if err := server.ListenAndServeTLS(*cert, *key); err != nil {
			log.Fatalf(err.Error())
		}

		break
	default:
		log.Fatal("select http or https")
	}
}
