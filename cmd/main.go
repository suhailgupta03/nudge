package main

import (
	"context"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"log"
	"nudge/activity"
	"nudge/actor"
	"nudge/internal/awslog"
	"nudge/internal/buflog"
	dbp "nudge/internal/database"
	"nudge/internal/database/user"
	"nudge/notify"
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

func initFlags() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	// Register the commandline flags.
	f.String("config", "config.yml", "path to config file")
	f.String("github.pem", "nudge.private-key.pem", "path to github pem file")
	if err := f.Parse(os.Args[1:]); err != nil {
		lo.Fatalf("error loading flags: %v", err)
	}
	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		lo.Fatalf("error loading config: %v", err)
	}
}

func main() {
	lo.Printf("TZ:%s", os.Getenv("TZ"))
	initFlags()
	if err := ko.Load(file.Provider(ko.String("config")), yaml.Parser()); err != nil {
		lo.Fatalf("error loading config from config.yml %v", err)
	}

	awsLogGroup := ko.String("aws.log_group")
	awsLogStream := ko.String("aws.log_stream")
	if len(awsLogStream) > 0 && len(awsLogGroup) > 0 {
		aws := awslog.AWSInit(awsLogGroup, awsLogStream)
		if aws.DoesLogGroupExist(awsLogGroup) && aws.DoesLogStreamExist(awsLogGroup, awsLogStream) {
			// Reinitialize the logger with AWS stream added
			lo = log.New(io.MultiWriter(os.Stdout, bufLog, awslog.New(2, *aws)), "",
				log.Ldate|log.Ltime|log.Lshortfile)
		} else {
			lo.Printf("Warn: Log group or log stream do not exist. Not auto creating them.")
		}
	} else {
		lo.Printf("Warn: aws cloudwatch details not present")
	}

	data, pemErr := os.ReadFile(ko.String("github.pem"))
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
	deps := new(WorkflowDependencies)
	deps.Activity = activity.Init(ko, database, lo)
	deps.ActorIdentifier = new(actor.Actor)
	deps.NotificationHours = new(notify.BusinessHours)
	deps.User = user.Init(database)
	deps.NotificationDays = &notify.NotificationDays{Lo: lo}
	Workflow(*deps)
	go func() {
		for {
			select {
			case <-ticker.C:
				Workflow(*deps)
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
