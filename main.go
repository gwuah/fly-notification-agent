package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	log    = logrus.New()
	logger = log.WithField("sys", "fly-notification-agent")

	machineID      = os.Getenv("FLY_MACHINE_ID")
	machineVersion = os.Getenv("FLY_MACHINE_VERSION")
	appName        = os.Getenv("FLY_APP_NAME")
)

type Event[T any] struct {
	Type string `json:"type"`
	Data T      `json:"data"`
}

func deliver(ctx context.Context, url string, events <-chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events:
			r, err := http.NewRequest("POST", url, bytes.NewBuffer(event))
			if err != nil {
				logger.WithError(err).Error("failed to deliver machine event to webhook")
			}
			r.Header.Add("Content-Type", "application/json")

			client := &http.Client{}
			res, err := client.Do(r)
			if err != nil {
				logger.WithError(err).Error("failed to deliver machine event to webhook")
			}
			res.Body.Close()
		}

	}
}

func generateEvent[T any](name string, e Event[T]) []byte {
	body, err := json.Marshal(e)
	if err != nil {
		logger.WithError(err).Errorf("failed to marshal %f event", name)
		return []byte("")
	}
	return body
}

func main() {
	app := &cli.App{
		Name:  "fly-notification-agent",
		Usage: "monitors vm state and sends events to specified webhook",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "webhook",
				Usage:    "webhook url",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			url := c.String("webhook")
			events := make(chan []byte, 5)
			sigChan := make(chan os.Signal, 1)

			signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				<-sigChan
				events <- generateEvent("machine_stopped", Event[int]{})
				cancel()
			}()

			logger = logger.WithField("machine", machineID).WithField("machine_version", machineVersion).WithField("app_id", appName).WithField("app_name", appName).WithField("url", url)
			logger.Info("Starting fly-notification-agent...")

			events <- generateEvent("machine_started", Event[int]{})

			return deliver(ctx, url, events)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	logger.Infof("fly-notification-agent is done")
}
