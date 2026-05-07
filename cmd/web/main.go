package main

import (
	"database/sql"
	"encoding/gob"
	"final-project/data"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	// connect to the DB
	db := initDb()

	_, err := GetEnv()
	if err != nil {
		log.Panic("can't load env file", err)
	}

	// create sessions
	session := initSession()

	// create loggers
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime)

	// create channels

	// create wg
	wg := sync.WaitGroup{}

	// set up the app config
	app := Config{
		Session:       session,
		DB:            db,
		Wait:          &wg,
		ErrorLog:      errorLog,
		InfoLog:       infoLog,
		Models:        data.New(db),
		ErrorChan:     make(chan error),
		ErrorChanDone: make(chan bool),
	}

	// set up mail
	app.Mailer = app.createMail()
	go app.listenForMail()

	// listen for signals
	go app.listenForShutdown()

	// listen for errors
	go app.listenForErrors()

	// listen for web connection
	app.serve()
}

func (app *Config) listenForErrors() {
	for {
		select {
		case err := <-app.ErrorChan:
			app.ErrorLog.Println(err)
		case <-app.ErrorChanDone:
			return
		}
	}
}

func (app *Config) serve() {
	// start http server

	port := os.Getenv("APP_PORT")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: app.routes(),
	}

	app.InfoLog.Printf("Starting web server on port=%s...", port)
	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func initDb() *sql.DB {
	conn := connectToDb()
	if conn == nil {
		log.Panic("can't connect to database")
	}

	return conn
}

func connectToDb() *sql.DB {
	count := 0
	dsn := os.Getenv("DSN")
	for {
		connection, err := openDb(dsn)
		if err != nil {
			log.Println("postgres not yet ready...", err)
		} else {
			log.Println("connect to database")
			return connection
		}

		if count > 10 {
			return nil
		}

		log.Println("Backing off for 1 second")
		time.Sleep(1 * time.Second)
		count++
	}
}

func openDb(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initSession() *scs.SessionManager {

	gob.Register(data.User{})

	// set up session
	session := scs.New()
	session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	return session
}

func initRedis() *redis.Pool {
	redisPool := &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}

	return redisPool
}

func (app *Config) listenForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.shutdown()
	os.Exit(0)
}

func (app *Config) shutdown() {
	// perform any cleanup tasks

	app.InfoLog.Println("Performing cleanup tasks...")

	// block until waitgroup is empty
	app.Wait.Wait()

	app.Mailer.DoneChan <- true
	app.ErrorChanDone <- true

	app.InfoLog.Println("closing channels and shutting down app...")
	close(app.Mailer.DoneChan)
	close(app.Mailer.ErrorChan)
	close(app.Mailer.MailerChan)
	close(app.ErrorChanDone)
	close(app.ErrorChan)
}

func (app *Config) createMail() Mail {
	// create channels
	errorChan := make(chan error)
	mailerChan := make(chan Message, 100)
	mailerDoneChan := make(chan bool)

	m := Mail{
		Domain:      "localhost",
		Host:        "localhost",
		Port:        1025,
		Encryption:  "none",
		FromName:    "Info",
		FromAddress: "info@mycompany.com",
		Wait:        app.Wait,
		ErrorChan:   errorChan,
		MailerChan:  mailerChan,
		DoneChan:    mailerDoneChan,
	}

	return m
}
