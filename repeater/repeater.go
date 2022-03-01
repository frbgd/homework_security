package repeater

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"homework_security/db"
	"homework_security/utils"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func ExecRepReq(respWriter http.ResponseWriter, request *http.Request) {
	dbConn, err := db.CreateNewDatabaseConnection()
	if err != nil {
		logrus.Warn("Can't connect to database")
		logrus.Error(err)
	}

	defer dbConn.Close()

	info := request.URL.Query()["id"]
	if len(info) < 1 {
		_, _ = fmt.Fprintf(respWriter,
			"Set id param to query in URL to repeat request\nVisit http://localhost/ for more info\n")
		return
	}

	if len(info) > 1 {
		_, _ = fmt.Fprintf(respWriter, "WARN: using first ID\n")
	}

	id, err := strconv.Atoi(info[0])
	if err != nil {
		logrus.Error(err)
	}

	req := dbConn.GetReqById(id)

	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	reqReader := bufio.NewReader(strings.NewReader(req.Request))
	buffer, err := http.ReadRequest(reqReader)
	if err != nil {
		logrus.Error(err)
	}

	httpReq, err := http.NewRequest(buffer.Method, req.Host, buffer.Body)
	if err != nil {
		logrus.Error(err)
	}

	utils.CopyHeaders(buffer.Header, httpReq.Header)

	resp, err := client.Do(httpReq)
	if err != nil {
		logrus.Error(err)
	}

	utils.CopyHeaders(resp.Header, respWriter.Header())
	respWriter.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(respWriter, resp.Body)
	_ = resp.Body.Close()

}
