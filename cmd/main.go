package main

import (
	"context"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"log"
	"nudge/internal/buflog"
	dbp "nudge/internal/database"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	log *log.Logger
	ko  *koanf.Koanf
	dbc *mongo.Client
	db  *mongo.Database
}

var (
	bufLog = buflog.New(5000)
	lo     = log.New(io.MultiWriter(os.Stdout, bufLog), "",
		log.Ldate|log.Ltime|log.Lshortfile)
	ko             = koanf.New(".")
	databaseClient *mongo.Client
	database       *mongo.Database
	dbCtx          context.Context
)

func main() {

	if err := ko.Load(file.Provider("config.yml"), yaml.Parser()); err != nil {
		lo.Fatalf("error loading config from config.yml %v", err)
	}

	data, pemErr := os.ReadFile("nudgetest.2023-04-14.private-key.pem")
	if pemErr != nil {
		lo.Fatalf("Failed to read the application pem file %v", pemErr)
	}

	ko.Set("app.private_key", string(data))

	databaseClient, dbCtx = initDatabaseConnection()
	database = databaseClient.Database(ko.String("mongo.database"))
	// Creates the database indexes if it does not exist
	dbp.SyncIndexes(database)
	defer databaseClient.Disconnect(dbCtx)

	app := &App{
		log: lo,
		ko:  ko,
		dbc: databaseClient,
		db:  database,
	}

	srv := initHTTPServer(app)

	ticker := time.NewTicker(time.Hour * ko.Duration("bot.next_check_in.time"))
	if ko.String("bot.next_check_in.unit") == "m" {
		ticker = time.NewTicker(time.Minute * ko.Duration("bot.next_check_in.time"))
	}

	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				Workflow()
				break
			case <-quit:
				ticker.Stop()
				break
			}
		}
	}()

	// Wait for the reload signal with a callback to gracefully shut down resources.
	// The `wait` channel is passed to awaitReload to wait for the callback to finish
	// within N seconds, or do a force reload.
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGHUP)

	closerWait := make(chan bool)
	<-awaitReload(sigChan, closerWait, func() {
		// Stop the HTTP server.
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		srv.Shutdown(ctx)

		// Signal the close.
		closerWait <- true
	})

}
