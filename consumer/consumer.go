package consumer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/swfrench/nginx-log-consumer/exporter"
	"github.com/swfrench/nginx-log-consumer/tailer"
)

const (
	ISO8601 = "2006-01-02T15:04:05-07:00"
)

type logLine struct {
	Time   string
	Status string
}

type Consumer struct {
	Period   time.Duration
	tailer   tailer.TailerT
	exporter exporter.ExporterT
	stop     chan bool
}

func NewConsumer(period time.Duration, tailer tailer.TailerT, exporter exporter.ExporterT) *Consumer {
	return &Consumer{
		Period:   period,
		tailer:   tailer,
		exporter: exporter,
		stop:     make(chan bool, 1),
	}
}

func (c *Consumer) consumeBytes(b []byte) error {
	statusCounts := make(map[string]int64)

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		lineBytes := scanner.Bytes()

		line := &logLine{}

		err := json.Unmarshal(lineBytes, line)
		if err != nil {
			log.Printf("Error parsing log line: %v", err)
			continue
		}

		t, err := time.Parse(ISO8601, line.Time)
		if err != nil {
			log.Printf("Could not parse time %v: %v", line.Time, err)
			continue
		}

		if t.Before(c.exporter.GetResetTime()) {
			continue
		}

		if tot, ok := statusCounts[line.Status]; ok {
			statusCounts[line.Status] = 1 + tot
		} else {
			statusCounts[line.Status] = 1
		}
	}
	log.Printf("Captured status code counters: %v", statusCounts)

	return c.exporter.IncrementStatusCounts(statusCounts)
}

func (c *Consumer) Run() error {
	for {
		select {
		case <-time.After(c.Period):
		case <-c.stop:
			return nil
		}
		b, err := c.tailer.Next()
		if err != nil {
			return fmt.Errorf("Could not retrieve log content: %v", err)
		} else if err := c.consumeBytes(b); err != nil {
			return fmt.Errorf("Could not export log content: %v", err)
		}
	}
	return nil
}

func (c *Consumer) Stop() {
	c.stop <- true
}