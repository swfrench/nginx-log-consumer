package main

import (
	"flag"
	"log"
	"log/syslog"
	"time"

	"github.com/swfrench/nginx-log-consumer/consumer"
	"github.com/swfrench/nginx-log-consumer/exporter"
	"github.com/swfrench/nginx-log-consumer/tailer"

	"cloud.google.com/go/compute/metadata"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/monitoring/v3"
)

var (
	accessLogPath = flag.String("access_log_path", "", "Path to access log file.")

	logPollingPeriod = flag.Duration("log_polling_period", 30*time.Second, "Period between checks for new log lines.")

	rotationCheckPeriod = flag.Duration("rotation_check_period", time.Minute, "Idle period between log rotation checks.")

	useSyslog = flag.Bool("use_syslog", false, "If true, emit info logs to syslog.")

	useMetadataService = flag.Bool("use_metadata_service", true, "If true, use the GCE instance metadata service to fetch project id, instance name, and zone name.")

	defaultProjectID = flag.String("default_project_id", "", "Project ID to use when metadata service is disabled or OnGCE() returns false.")

	defaultInstanceName = flag.String("default_instance_name", "", "Instance name to use when metadata service is disabled or OnGCE() returns false.")

	defaultZoneName = flag.String("default_zone_name", "", "Zone name to use when metadata service is disabled or OnGCE() returns false.")

	createCustomMetrics = flag.Bool("create_custom_metrics", false, "If true, attempt to create custom metrics before starting logs consumption.")
)

func getMetadata() (string, map[string]string) {
	var projectID string
	var resourceLabels map[string]string

	if metadata.OnGCE() && *useMetadataService {
		// Get project, instance, zone from metadata.
		id, err := metadata.ProjectID()
		if err != nil {
			log.Fatalf("Could not retrieve project ID from metadata service: %v", err)
		}

		instance, err := metadata.InstanceName()
		if err != nil {
			log.Fatalf("Could not retrieve instance name from metadata service: %v", err)
		}

		zone, err := metadata.Zone()
		if err != nil {
			log.Fatalf("Could not retrieve zone name from metadata service: %v", err)
		}

		projectID = id
		resourceLabels = map[string]string{
			"instance_id": instance,
			"zone":        zone,
		}
	} else {
		if *defaultProjectID == "" {
			log.Fatalf("Metadata service is disabled or not available, but default_project_id is not set.")
		}
		if *defaultInstanceName == "" {
			log.Fatalf("Metadata service is disabled or not available, but default_instance_name is not set.")
		}
		if *defaultZoneName == "" {
			log.Fatalf("Metadata service is disabled or not available, but default_zone_name is not set.")
		}

		projectID = *defaultProjectID
		resourceLabels = map[string]string{
			"instance_id": *defaultInstanceName,
			"zone":        *defaultZoneName,
		}
	}
	return projectID, resourceLabels
}

func main() {
	flag.Parse()

	if *useSyslog {
		w, err := syslog.New(syslog.LOG_INFO, "nginx_log_consumer")
		if err != nil {
			log.Fatalf("Could not create syslog writer: %v", err)
		}
		log.SetOutput(w)
	}

	t, err := tailer.NewTailer(*accessLogPath, *rotationCheckPeriod)
	if err != nil {
		log.Fatalf("Could not create tailer for %s: %v", *accessLogPath, err)
	}

	ctx := context.Background()
	client, err := google.DefaultClient(ctx, monitoring.MonitoringScope)
	if err != nil {
		log.Fatalf("Could not create Google API client: %v", err)
	}

	monitoringService, err := monitoring.New(client)
	if err != nil {
		log.Fatalf("Could not create Cloud Monitoring client: %v", err)
	}

	projectID, resourceLabels := getMetadata()

	log.Printf("Creating GCM exporter for project %s; resource: %v", projectID, resourceLabels)

	e := exporter.NewCloudMonitoringExporter(monitoringService, projectID, resourceLabels)

	if *createCustomMetrics {
		if err := e.CreateMetrics(); err != nil {
			log.Fatalf("Failed to create custom metrics: %v", err)
		}
	}

	c := consumer.NewConsumer(*logPollingPeriod, t, e)

	log.Printf("Starting consumer for %s", *accessLogPath)

	if err := c.Run(); err != nil {
		log.Fatalf("Failure consuming logs: %v", err)
	}
}
