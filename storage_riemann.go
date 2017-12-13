package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blockchaindev/goryman"
)

// RiemannConfig describes the YAML-provided configuration for a Riemann
// storage backend
type RiemannConfig struct {
	Host      string   `yaml:"host"`
	Port      int      `yaml:"port"`
	Namespace string   `yaml:"metric-namespace,omitempty"`
	Tags      []string `yaml:"tags,omitempty"`
}

// RiemannStorage holds the configuration for a Riemann storage backend
type RiemannStorage struct {
	Namespace string
	Tags      []string
	Client    *goryman.GorymanClient
}

// NewRiemannStorage sets up a new Riemann storage backend
func NewRiemannStorage(c *Config) (RiemannStorage, error) {
	r := RiemannStorage{}

	r.Namespace = c.Storage.Riemann.Namespace
	r.Tags = c.Storage.Riemann.Tags

	r.Client = goryman.NewGorymanClient(fmt.Sprint(c.Storage.Riemann.Host, ":", c.Storage.Riemann.Port))
	var err error
	for i := 0; true; i++ {
		log.Printf("Trying to connect to riemann, attempt %d", i)
		err = r.Client.Connect()
		if err != nil {
			log.Printf("Could not connect to Riemann server: %v", err)
		} else {
			err = nil
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}

	if err != nil {
		return r, err
	}

	return r, nil
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to a Prometheus pushgateway
func (r RiemannStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	// Riemann storage supports both metrics and events, so we'll initialize both channels
	metricChan := make(chan Metric, 10)
	eventChan := make(chan Event, 10)

	// Start processing the metrics we receive
	go r.processMetricsAndEvents(ctx, wg, metricChan, eventChan)

	return metricChan, eventChan
}

func (r RiemannStorage) processMetricsAndEvents(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric, echan <-chan Event) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-mchan:
			err := r.sendMetric(m)
			if err != nil {
				log.Println(err)
			}
		case e := <-echan:
			err := r.sendEvent(e)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request received. Cancelling metrics processor.")
			return
		}
	}
}

// sendMetric sends a metric value to Riemann
func (r RiemannStorage) sendMetric(m Metric) error {
	var metricName string

	if r.Namespace == "" {
		metricName = fmt.Sprintf("crabby.%v", m.Name)
	} else {
		metricName = fmt.Sprintf("%v.%v", r.Namespace, m.Name)
	}

	ev := &goryman.Event{
		Service: metricName,
		Metric:  m.Value,
		Tags:    r.Tags,
		Attributes: map[string]string{
			"url": m.Url,
		},
	}

	err := r.Client.SendEvent(ev)
	if err != nil {
		return err
	}

	return nil
}

// sendEvent sends an event to Riemann
func (r RiemannStorage) sendEvent(e Event) error {
	var eventName string
	var state string

	if r.Namespace == "" {
		eventName = fmt.Sprintf("crabby.%v", e.Name)
	} else {
		eventName = fmt.Sprintf("%v.%v", r.Namespace, e.Name)
	}

	if (e.ServerStatus < 400) && (e.ServerStatus > 0) {
		state = "ok"
	} else {
		state = "critical"
	}

	ev := &goryman.Event{
		Service: eventName,
		State:   state,
		Tags:    r.Tags,
		Attributes: map[string]string{
			"url": e.Url,
		},
	}

	err := r.Client.SendEvent(ev)
	if err != nil {
		return err
	}

	return nil
}
