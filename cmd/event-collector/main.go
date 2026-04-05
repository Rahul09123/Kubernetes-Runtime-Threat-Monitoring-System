package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rahulraman/kubernetes-runtime-threat-monitoring-system/internal/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue, err := common.NewQueue(env("NATS_URL", "nats://nats:4222"))
	if err != nil {
		log.Fatalf("queue setup failed: %v", err)
	}
	defer queue.Close()

	metrics := common.NewServiceMetrics("event-collector")
	go serveHTTP(env("HTTP_ADDR", ":8080"))

	clientset, err := kubeClient()
	if err != nil {
		log.Fatalf("kubernetes client setup failed: %v", err)
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)
	podInformer := factory.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod := obj.(*corev1.Pod)
			publishEvent(ctx, queue, metrics, common.PodEvent{
				Type:      "POD_CREATED",
				Namespace: pod.Namespace,
				Pod:       pod.Name,
				Reason:    "Created",
				Message:   "Pod created",
				Labels:    pod.Labels,
				Timestamp: time.Now().UTC(),
			})
		},
		UpdateFunc: func(oldObj, newObj any) {
			oldPod := oldObj.(*corev1.Pod)
			newPod := newObj.(*corev1.Pod)
			for i := range newPod.Status.ContainerStatuses {
				if i >= len(oldPod.Status.ContainerStatuses) {
					continue
				}
				if newPod.Status.ContainerStatuses[i].RestartCount > oldPod.Status.ContainerStatuses[i].RestartCount {
					publishEvent(ctx, queue, metrics, common.PodEvent{
						Type:      "POD_RESTART",
						Namespace: newPod.Namespace,
						Pod:       newPod.Name,
						Reason:    "ContainerRestart",
						Message:   "Pod container restart detected",
						Labels:    newPod.Labels,
						Timestamp: time.Now().UTC(),
					})
					return
				}
			}
		},
		DeleteFunc: func(obj any) {
			pod := obj.(*corev1.Pod)
			publishEvent(ctx, queue, metrics, common.PodEvent{
				Type:      "POD_DELETED",
				Namespace: pod.Namespace,
				Pod:       pod.Name,
				Reason:    "Deleted",
				Message:   "Pod deleted",
				Labels:    pod.Labels,
				Timestamp: time.Now().UTC(),
			})
		},
	})

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, podInformer.HasSynced) {
		log.Fatal("failed to sync pod informer cache")
	}

	log.Println("event collector started")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	close(stopCh)
}

func publishEvent(ctx context.Context, queue *common.Queue, metrics *common.ServiceMetrics, event common.PodEvent) {
	if strings.HasPrefix(event.Namespace, "kube-") {
		return
	}
	if err := queue.Publish(ctx, "pods.events", event); err != nil {
		log.Printf("publish event failed: %v", err)
		return
	}
	metrics.EventsProcessed.Inc()
}

func kubeClient() (*kubernetes.Clientset, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return kubernetes.NewForConfig(cfg)
	}
	kubeconfig := env("KUBECONFIG", os.ExpandEnv("$HOME/.kube/config"))
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
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
