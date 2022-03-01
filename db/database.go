package db

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"os"
)

type Database struct {
	dbConn *sql.DB
}

const dbName = "proxy.db"
const schema = `
	CREATE TABLE "requests" (
	"id"	INTEGER NOT NULL,
	"host"	TEXT,
	"request"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
`

type Request struct {
	Host    string
	Request string
	Id      int
}

func CreateNewDatabaseConnection() (*Database, error) {
	var needToInitDB bool
	if _, err := os.Stat(dbName); errors.Is(err, os.ErrNotExist) {
		needToInitDB = true
	}
	dbConn, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}
	logrus.Debug("DB connection is open")
	if needToInitDB {
		_, err := dbConn.Exec(schema)
		if err != nil {
			return nil, err
		}
		logrus.Debug("Initialized the database")
	}
	return &Database{
		dbConn: dbConn,
	}, nil
}

func (db *Database) InsertRequest(dbReq Request) {
	_, err := db.dbConn.Exec(
		"INSERT INTO requests (host, request) VALUES ($1, $2)",
		dbReq.Host,
		dbReq.Request,
	)

	if err != nil {
		logrus.Warn("Can't save request to database")
		logrus.Error(err)
	}
}

func (db *Database) GetRequestList() ([]Request, error) {
	rows, err := db.dbConn.Query(
		"SELECT * FROM requests",
	)

	if err != nil {
		return nil, err
	}

	requests := make([]Request, 0)
	for rows.Next() {
		req := Request{}
		err := rows.Scan(&req.Id, &req.Host, &req.Request)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	err = rows.Close()
	if err != nil {
		return nil, err
	}
	return requests, nil
}

func (db *Database) GetReqById(id int) Request {
	req := Request{}
	row := db.dbConn.QueryRow(
		"SELECT * FROM requests WHERE id=$1", id)
	err := row.Scan(&req.Id, &req.Host, &req.Request)
	if err != nil {
		logrus.Warn("Can't get data from database")
		logrus.Error(err)
	}

	return req
}

func (db *Database) Close() {
	err := db.dbConn.Close()
	if err != nil {
		logrus.Warn(err)
	}
}
