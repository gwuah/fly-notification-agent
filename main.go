package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

type Data struct {
	MachineID string `json:"machine_id"`
	AppName   string `json:"app_name"`
	At        int64  `json:"at"`
}

type Event struct {
	Type string `json:"type"`
	Data Data   `json:"data"`
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
				events <- generateEvent("machine_stopped")
				cancel()
			}()

			logger = logger.WithField("machine", machineID).WithField("machine_version", machineVersion).WithField("app_id", appName).WithField("app_name", appName).WithField("url", url)
			logger.Info("Starting fly-notification-agent...")

			events <- generateEvent("machine_started")

			go oomChecker(ctx, events)

			return deliverWebhooks(ctx, url, events)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	logger.Infof("fly-notification-agent is done")
}

func generateEvent(name string) []byte {
	e := Event{
		Type: name,
		Data: Data{
			MachineID: machineID,
			AppName:   appName,
			At:        time.Now().Unix(),
		},
	}
	body, err := json.Marshal(e)
	if err != nil {
		logger.WithError(err).Errorf("failed to marshal %f event", name)
		return []byte("")
	}
	return body
}

func oomChecker(ctx context.Context, events chan []byte) {
	reportedOOMs := map[string]struct{}{}
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			file, err := os.Open("/dev/kmsg")
			if err != nil {
				logger.WithError(err).Errorf("failed to open file")
				continue
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				text := scanner.Text()
				if strings.Contains(text, "Killed process") {
					if _, ok := reportedOOMs[text]; !ok {
						events <- generateEvent("oom")
						reportedOOMs[text] = struct{}{}
					}
				}
			}

			file.Close()

			if err := scanner.Err(); err != nil {
				logger.WithError(err).Errorf("failed to scan file")
			}

		}
	}
}

func deliverWebhooks(ctx context.Context, url string, events <-chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events:
			if len(event) == 0 {
				continue
			}

			r, err := http.NewRequest("POST", url, bytes.NewBuffer(event))
			if err != nil {
				logger.WithError(err).Error("failed to deliver event")
			}
			r.Header.Add("Content-Type", "application/json")

			client := &http.Client{}
			res, err := client.Do(r)
			if err != nil {
				logger.WithError(err).Error("failed to deliver event")
				continue
			}
			res.Body.Close()
		}

	}
}
