package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"

	"github.com/rahulraman/kubernetes-runtime-threat-monitoring-system/internal/common"
)

func main() {
	queue, err := common.NewQueue(env("NATS_URL", "nats://nats:4222"))
	if err != nil {
		log.Fatalf("queue setup failed: %v", err)
	}
	defer queue.Close()

	metrics := common.NewServiceMetrics("alert-manager")
	go serveHTTP(env("HTTP_ADDR", ":8082"))

	_, err = queue.Subscribe("threats.alerts", func(payload []byte) {
		var alert common.ThreatAlert
		if err := json.Unmarshal(payload, &alert); err != nil {
			log.Printf("decode alert failed: %v", err)
			return
		}
		metrics.EventsProcessed.Inc()
		metrics.AlertsRaised.WithLabelValues(alert.Severity).Inc()
		if err := dispatch(alert); err != nil {
			log.Printf("dispatch failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	log.Println("alert manager started")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func dispatch(alert common.ThreatAlert) error {
	var errSlack error
	var errEmail error

	if webhook := os.Getenv("SLACK_WEBHOOK_URL"); webhook != "" {
		errSlack = sendSlack(webhook, alert)
	}
	if os.Getenv("SMTP_HOST") != "" {
		errEmail = sendEmail(alert)
	}

	if errSlack != nil && errEmail != nil {
		return fmt.Errorf("slack and email failed: %v | %v", errSlack, errEmail)
	}
	if os.Getenv("SLACK_WEBHOOK_URL") == "" && os.Getenv("SMTP_HOST") == "" {
		log.Printf("no notification backend configured, alert logged only: %+v", alert)
	}
	return nil
}

func sendSlack(webhook string, alert common.ThreatAlert) error {
	payload := map[string]string{
		"text": fmt.Sprintf("[%s] %s (ns=%s pod=%s source=%s)", alert.Severity, alert.Summary, alert.Namespace, alert.Pod, alert.Source),
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}
	return nil
}

func sendEmail(alert common.ThreatAlert) error {
	host := os.Getenv("SMTP_HOST")
	port := env("SMTP_PORT", "587")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := env("SMTP_FROM", user)
	to := os.Getenv("SMTP_TO")
	if to == "" {
		return fmt.Errorf("SMTP_TO not set")
	}

	msg := []byte(fmt.Sprintf("Subject: Kubernetes Threat Alert\r\n\r\nSeverity: %s\nCategory: %s\nNamespace: %s\nPod: %s\nSummary: %s\nSource: %s\n", alert.Severity, alert.Category, alert.Namespace, alert.Pod, alert.Summary, alert.Source))
	addr := fmt.Sprintf("%s:%s", host, port)
	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func serveHTTP(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", common.MetricsHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
