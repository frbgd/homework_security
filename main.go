package main

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"homework_security/db"
	"homework_security/repeater"
	"homework_security/utils"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"
)

const reqDumpErr = "Request dump error: "
const dbConnectErr = "Can't connect to database"

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

func handleHTTPS(w http.ResponseWriter, r *http.Request, dbConn *db.Database) {
	connectID := uuid.New()
	fmt.Println("HTTPS Connect", connectID, r.Method, r.URL)

	reqDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		logrus.Warn(reqDumpErr, err.Error())
	}

	dbReq := db.Request{
		Host:    "https://" + r.Host,
		Request: string(reqDump),
	}

	dbConn.InsertRequest(dbReq)

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
	dbConn.Close()

	fmt.Println("HTTPS open transfer", connectID)
}

func handleHTTP(w http.ResponseWriter, r *http.Request, dbConn *db.Database) {
	connectID := uuid.New()
	fmt.Println("HTTP Connect", connectID, r.Method, r.URL)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	r.RequestURI = ""

	reqDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		logrus.Warn(reqDumpErr, err.Error())
	}

	dbReq := db.Request{
		Host:    "http://" + r.Host,
		Request: string(reqDump),
	}
	dbConn.InsertRequest(dbReq)

	resp, err := client.Do(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal("handleHTTP:", err)
	}
	defer resp.Body.Close()

	fmt.Println("HTTP Response", connectID, resp.Status)

	utils.CopyHeaders(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	dbConn.Close()

	fmt.Println("HTTP Close", connectID)
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dbConn, err := db.CreateNewDatabaseConnection()
	if err != nil {
		logrus.Fatal(dbConnectErr, err.Error())
	}

	if r.Method == http.MethodConnect {
		handleHTTPS(w, r, dbConn)
	} else {
		handleHTTP(w, r, dbConn)
	}
}

func main() {
	proxyPort := "8080"
	if port := os.Getenv("PROXY_PORT"); len(port) != 0 {
		proxyPort = port
	}
	dashboardPort := "80"
	if port := os.Getenv("DASHBOARD_PORT"); len(port) != 0 {
		dashboardPort = port
	}

	server := &http.Server{
		Addr:    ":" + proxyPort,
		Handler: http.HandlerFunc(ServeHTTP),
	}

	http.HandleFunc("/req", repeater.ExecRepReq)

	go http.ListenAndServe(":"+dashboardPort, nil)
	logrus.Fatal(server.ListenAndServe())
}
