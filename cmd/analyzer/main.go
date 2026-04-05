package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rahulraman/kubernetes-runtime-threat-monitoring-system/internal/common"
)

type FalcoWebhook struct {
	Output       string                 `json:"output"`
	Priority     string                 `json:"priority"`
	Rule         string                 `json:"rule"`
	Time         string                 `json:"time"`
	OutputFields map[string]interface{} `json:"output_fields"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue, err := common.NewQueue(env("NATS_URL", "nats://nats:4222"))
	if err != nil {
		log.Fatalf("queue setup failed: %v", err)
	}
	defer queue.Close()

	metrics := common.NewServiceMetrics("analyzer")
	go serveHTTP(queue, metrics)

	_, err = queue.Subscribe("pods.events", func(payload []byte) {
		var event common.PodEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Printf("decode pod event failed: %v", err)
			return
		}
		metrics.EventsProcessed.Inc()
		if alert, ok := detectFromPodEvent(event); ok {
			metrics.AlertsRaised.WithLabelValues(alert.Severity).Inc()
			if err := queue.Publish(ctx, "threats.alerts", alert); err != nil {
				log.Printf("publish alert failed: %v", err)
			}
		}
	})
	if err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	log.Println("analyzer started")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func detectFromPodEvent(event common.PodEvent) (common.ThreatAlert, bool) {
	if event.Type == "POD_RESTART" {
		return common.ThreatAlert{
			Severity:  "medium",
			Category:  "anomaly",
			Source:    "pod-event",
			Namespace: event.Namespace,
			Pod:       event.Pod,
			Summary:   "Abnormal pod restart detected",
			Detected:  time.Now().UTC(),
		}, true
	}

	if v, ok := event.Labels["security.privileged"]; ok && strings.EqualFold(v, "true") {
		return common.ThreatAlert{
			Severity:  "high",
			Category:  "privilege-escalation",
			Source:    "pod-event",
			Namespace: event.Namespace,
			Pod:       event.Pod,
			Summary:   "Potential privileged pod configuration detected",
			Detected:  time.Now().UTC(),
		}, true
	}

	return common.ThreatAlert{}, false
}

func serveHTTP(queue *common.Queue, metrics *common.ServiceMetrics) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", common.MetricsHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/falco", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var event FalcoWebhook
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid payload"))
			return
		}

		severity := strings.ToLower(event.Priority)
		if severity == "" {
			severity = "high"
		}

		alert := common.ThreatAlert{
			Severity:  severity,
			Category:  "runtime-threat",
			Source:    "falco",
			Namespace: asString(event.OutputFields["k8s.ns.name"]),
			Pod:       asString(event.OutputFields["k8s.pod.name"]),
			Summary:   event.Output,
			Detected:  time.Now().UTC(),
		}

		metrics.AlertsRaised.WithLabelValues(alert.Severity).Inc()
		if err := queue.Publish(r.Context(), "threats.alerts", alert); err != nil {
			log.Printf("publish falco alert failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("failed to publish alert"))
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("alert ingested"))
	})

	addr := env("HTTP_ADDR", ":8081")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
