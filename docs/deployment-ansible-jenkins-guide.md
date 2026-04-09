# Deployment, Ansible, and Jenkins Guide

This document explains the tools used in this repository, how the runtime and deployment flow works, what each microservice does, and how the deployment, Ansible, and Jenkins files work line by line.

## 1. Tools Used and Why

### Kubernetes
Used to run the application as separate services in a cluster.

Why it is used:
- It gives isolated runtime environments for each service.
- It lets the system scale and restart services independently.
- It provides stable service discovery through Kubernetes Services.
- It allows the monitoring stack to run alongside the app stack.

### Kustomize
Used through `kubectl apply -k` for the manifests in `deployments/k8s/`.

Why it is used:
- It keeps shared configuration in one place.
- It makes the base resources easy to apply together.
- It avoids manual editing of many YAML files during deploys.

### Docker
Used to build the images for the Go services.

Why it is used:
- It packages each binary and its runtime into a repeatable artifact.
- It lets Jenkins build and publish images in a predictable way.
- It matches what Minikube and Kubernetes need to run the services.

### Jenkins
Used as the CI/CD orchestrator.

Why it is used:
- It runs tests before deployment.
- It builds images and scans them with Trivy.
- It pushes images to the registry.
- It deploys the manifests and waits for rollout completion.

### Ansible
Used to provide a simple local deployment automation path.

Why it is used:
- It installs required tools on macOS when needed.
- It applies the Kubernetes manifests without manual repetition.
- It gives a repeatable setup-and-deploy flow outside Jenkins.

### kubectl
Used for cluster operations.

Why it is used:
- It applies the manifests.
- It updates image tags in deployments.
- It checks rollout status and cluster state.

### Trivy
Used in Jenkins to scan images.

Why it is used:
- It blocks builds with HIGH or CRITICAL vulnerabilities.
- It adds a security gate before push or deploy.

### NATS
Used as the internal event bus.

Why it is used:
- It decouples event production from event consumption.
- Event Collector can publish without knowing who consumes the event.
- Analyzer and Alert Manager communicate asynchronously.

### Prometheus
Used to scrape service metrics.

Why it is used:
- It captures throughput and alert counters.
- It provides the operational data behind monitoring.

### Alert Manager Web UI
Used as the frontend for the monitoring system.

Why it is used:
- It shows the latest alerts and summary cards.
- It gives operators a direct UI for the pipeline output.
- It is the main frontend for this project.

### Go
Used to implement the services and their Kubernetes integrations.

Why it is used:
- It is the language of the services in this repository.
- It works well with client-go, Prometheus client libraries, and NATS.

## 2. How Everything Works

### High-level runtime flow
1. Kubernetes starts all application pods in the `krtms` namespace.
2. NATS runs as the message bus.
3. Event Collector watches pod lifecycle events in the cluster.
4. Event Collector publishes normalized pod events to NATS on `pods.events`.
5. Analyzer subscribes to `pods.events` and applies threat-detection rules.
6. Analyzer can also accept Falco webhook calls at `/falco`.
7. Analyzer publishes alert objects to NATS on `threats.alerts`.
8. Alert Manager subscribes to `threats.alerts`.
9. Alert Manager stores alerts in memory, exposes `/api/alerts`, and renders the web dashboard.
10. Prometheus scrapes the `/metrics` endpoints from the services.
11. The alert-manager frontend reads `/api/alerts` and displays the latest detections.

### Deployment flow
1. Jenkins builds the Go binaries into Docker images.
2. Jenkins scans the images with Trivy.
3. Jenkins pushes the images when the branch is `main`.
4. Jenkins applies the Kubernetes manifests.
5. Jenkins updates the deployment image tags to the current build tag.
6. Jenkins waits for rollout completion.
7. Jenkins optionally runs the smoke test to prove the alert pipeline.

### Local automation flow
1. Ansible optionally installs `kubectl` and `helm` on macOS.
2. Ansible applies the base manifests.
3. Ansible applies the monitoring manifests.
4. You can then inspect pods, services, Prometheus, and Alert Manager.

### What the deployment stack provides
- A namespace for isolation.
- NATS for asynchronous messaging.
- Event Collector to capture pod activity.
- Analyzer to detect suspicious behavior.
- Alert Manager to store and display alerts.
- Prometheus to collect metrics.
- Alert Manager UI to show live operational output.

## 2.1 What Each Microservice Does

### Event Collector
The Event Collector watches Kubernetes pod events in the cluster.

What it does:
- Uses `client-go` informers to watch pod add, update, and delete events.
- Converts raw Kubernetes events into the shared `PodEvent` model.
- Publishes those events to NATS on the `pods.events` subject.
- Exposes `/healthz` and `/metrics` so its status can be monitored.

Why it exists:
- It isolates Kubernetes event capture from threat analysis.
- It makes the raw cluster event stream reusable by other consumers.

### Analyzer
The Analyzer consumes pod events and turns suspicious activity into alerts.

What it does:
- Subscribes to `pods.events` from NATS.
- Applies detection rules like restart spikes and privileged labels.
- Accepts Falco webhook alerts at `POST /falco`.
- Publishes `ThreatAlert` objects to NATS on `threats.alerts`.
- Exposes `/healthz` and `/metrics` for observability.

Why it exists:
- It centralizes the threat-detection logic.
- It lets both Kubernetes behavior and Falco runtime findings feed the same alert pipeline.

### Alert Manager
The Alert Manager receives alerts and presents them to the user.

What it does:
- Subscribes to `threats.alerts` from NATS.
- Stores recent alerts in memory.
- Serves the frontend dashboard at `/`.
- Serves the JSON API at `/api/alerts`.
- Sends Slack or email notifications when configured.
- Exposes `/healthz` and `/metrics` for monitoring.

Why it exists:
- It separates alert production from alert presentation and delivery.
- It gives operators both a UI and an API for recent detections.

### NATS
NATS is the messaging layer between the services.

What it does:
- Carries pod events from Event Collector to Analyzer.
- Carries threat alerts from Analyzer to Alert Manager.
- Keeps the services loosely coupled.

Why it exists:
- It allows the pipeline to keep working even if the producer and consumer are not tightly synchronized.

### Prometheus
Prometheus collects service metrics.

What it does:
- Scrapes the `/metrics` endpoints from all services.
- Stores counters such as events processed and alerts raised.
- Supplies data for the monitoring view.

Why it exists:
- It provides the operational data behind the monitoring view.

### Alert Manager UI
The Alert Manager UI is the frontend for the project.

What it does:
- Serves the dashboard at `/`.
- Shows recent alerts from `/api/alerts`.
- Displays alert counts, top namespaces, categories, and pods.
- Updates live by polling the backend API.

Why it exists:
- It is the operator-facing frontend for the monitoring system.
- It is the UI for this project.

## 3. Line-by-Line Explanation

### 3.1 Jenkinsfile

File: `Jenkinsfile`

```groovy
pipeline {
  agent any

  environment {
    REGISTRY = 'ghcr.io/rahul09123'
    TAG = "${env.BUILD_NUMBER}"
    TRIVY_IMAGE = 'ghcr.io/aquasecurity/trivy:0.51.4'
    RUN_SMOKE_TEST = "${env.RUN_SMOKE_TEST ?: 'false'}"
    K8S_NAMESPACE = "${env.K8S_NAMESPACE ?: 'krtms'}"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Go Test') {
      steps {
        sh 'go test ./...'
      }
    }

    stage('Build Images') {
      steps {
        sh 'docker build -f deployments/docker/Dockerfile.event-collector -t $REGISTRY/event-collector:$TAG .'
        sh 'docker build -f deployments/docker/Dockerfile.analyzer -t $REGISTRY/analyzer:$TAG .'
        sh 'docker build -f deployments/docker/Dockerfile.alert-manager -t $REGISTRY/alert-manager:$TAG .'
      }
    }

    stage('Trivy Scan') {
      steps {
        sh '''
          set -euo pipefail
          docker run --rm -v /var/run/docker.sock:/var/run/docker.sock "$TRIVY_IMAGE" image --severity HIGH,CRITICAL --exit-code 1 "$REGISTRY/event-collector:$TAG"
          docker run --rm -v /var/run/docker.sock:/var/run/docker.sock "$TRIVY_IMAGE" image --severity HIGH,CRITICAL --exit-code 1 "$REGISTRY/analyzer:$TAG"
          docker run --rm -v /var/run/docker.sock:/var/run/docker.sock "$TRIVY_IMAGE" image --severity HIGH,CRITICAL --exit-code 1 "$REGISTRY/alert-manager:$TAG"
        '''
      }
    }

    stage('Push Images') {
      when {
        expression {
          return env.BRANCH_NAME == 'main' || env.GIT_BRANCH == 'main' || env.GIT_BRANCH == 'origin/main'
        }
      }
      steps {
        withCredentials([usernamePassword(credentialsId: 'registry-creds', passwordVariable: 'REG_PASS', usernameVariable: 'REG_USER')]) {
          sh 'echo $REG_PASS | docker login ghcr.io -u $REG_USER --password-stdin'
          sh 'docker push $REGISTRY/event-collector:$TAG'
          sh 'docker push $REGISTRY/analyzer:$TAG'
          sh 'docker push $REGISTRY/alert-manager:$TAG'
        }
      }
    }

    stage('Deploy to Kubernetes') {
      when {
        expression {
          return env.BRANCH_NAME == 'main' || env.GIT_BRANCH == 'main' || env.GIT_BRANCH == 'origin/main'
        }
      }
      steps {
        sh '''
          set -euo pipefail
          kubectl apply -k deployments/k8s/base
          kubectl apply -k deployments/k8s/monitoring
          kubectl set image -n "$K8S_NAMESPACE" deploy/event-collector event-collector="$REGISTRY/event-collector:$TAG"
          kubectl set image -n "$K8S_NAMESPACE" deploy/analyzer analyzer="$REGISTRY/analyzer:$TAG"
          kubectl set image -n "$K8S_NAMESPACE" deploy/alert-manager alert-manager="$REGISTRY/alert-manager:$TAG"
          kubectl rollout status deploy/event-collector -n "$K8S_NAMESPACE" --timeout=180s
          kubectl rollout status deploy/analyzer -n "$K8S_NAMESPACE" --timeout=180s
          kubectl rollout status deploy/alert-manager -n "$K8S_NAMESPACE" --timeout=180s
          kubectl rollout status deploy/prometheus -n "$K8S_NAMESPACE" --timeout=180s
        '''
      }
    }

    stage('Smoke Test') {
      when {
        expression {
          return (env.BRANCH_NAME == 'main' || env.GIT_BRANCH == 'main' || env.GIT_BRANCH == 'origin/main') && env.RUN_SMOKE_TEST?.toBoolean()
        }
      }
      steps {
        sh '''
          set -euo pipefail
          kubectl version --client
          NAMESPACE="$K8S_NAMESPACE" make smoke
        '''
      }
    }
  }
}
```

Explanation by line or block:
- `pipeline {` starts a Jenkins declarative pipeline.
- `agent any` tells Jenkins to run this on any available agent.
- `environment {` defines reusable environment variables.
- `REGISTRY = 'ghcr.io/rahul09123'` sets the container registry target.
- `TAG = "${env.BUILD_NUMBER}"` uses the Jenkins build number as the image tag.
- `TRIVY_IMAGE = ...` sets the Trivy image used for scans.
- `RUN_SMOKE_TEST = ...` lets the job enable or disable smoke testing.
- `K8S_NAMESPACE = ...` lets the deploy target namespace be configurable.
- `stage('Checkout')` pulls the repository source.
- `checkout scm` checks out the current pipeline source control revision.
- `stage('Go Test')` runs the Go test suite.
- `sh 'go test ./...'` tests all Go packages.
- `stage('Build Images')` builds the three service images.
- Each `docker build` command builds one service image from its Dockerfile.
- `stage('Trivy Scan')` runs vulnerability checks on the built images.
- The Trivy commands fail the build on HIGH and CRITICAL issues.
- `stage('Push Images')` pushes images to the registry only on the `main` branch.
- `withCredentials(...)` injects registry credentials securely.
- `docker login` authenticates to the registry.
- `stage('Deploy to Kubernetes')` applies and updates the live cluster.
- `kubectl apply -k deployments/k8s/base` applies the application stack.
- `kubectl apply -k deployments/k8s/monitoring` applies Prometheus.
- `kubectl set image ...` updates each deployment to the new build tag.
- `kubectl rollout status ...` waits until each deployment becomes healthy.
- `stage('Smoke Test')` runs only when both deploy and smoke flags are enabled.
- `kubectl version --client` confirms kubectl is available in the Jenkins agent.
- `NAMESPACE="$K8S_NAMESPACE" make smoke` runs the repository smoke test against the configured namespace.

### 3.2 Ansible Inventory

File: `ansible/inventory/hosts.ini`

```ini
[k8s_master]
127.0.0.1 ansible_connection=local
```

Explanation:
- `[k8s_master]` defines a host group named `k8s_master`.
- `127.0.0.1` means the playbook runs against the local machine.
- `ansible_connection=local` tells Ansible not to SSH; it uses the local shell.

### 3.3 Ansible Playbook

File: `ansible/playbooks/setup_and_deploy.yml`

```yaml
---
- name: Setup tooling and deploy KRTMS
  hosts: k8s_master
  become: false
  vars:
    project_dir: "{{ playbook_dir }}/../.."
  tasks:
    - name: Ensure Homebrew packages are installed (macOS)
      ansible.builtin.command: "brew install kubectl helm"
      changed_when: false
      when: ansible_system == 'Darwin'

    - name: Apply base manifests
      ansible.builtin.command: "kubectl apply -k {{ project_dir }}/deployments/k8s/base"
      changed_when: false

    - name: Apply monitoring manifests
      ansible.builtin.command: "kubectl apply -k {{ project_dir }}/deployments/k8s/monitoring"
      changed_when: false

    - name: Display services
      ansible.builtin.command: "kubectl get svc -n krtms"
      changed_when: false
```

Explanation by line or block:
- `---` starts the YAML document.
- `- name: Setup tooling and deploy KRTMS` names the play.
- `hosts: k8s_master` runs the play against the local inventory group.
- `become: false` means no privilege escalation is used.
- `vars:` defines playbook variables.
- `project_dir: "{{ playbook_dir }}/../.."` resolves the repository root from the playbook path.
- `tasks:` begins the task list.
- `Ensure Homebrew packages are installed (macOS)` installs local CLI tools if the host is macOS.
- `ansible.builtin.command: "brew install kubectl helm"` runs the Homebrew install command.
- `changed_when: false` prevents Ansible from reporting the task as changed every time.
- `when: ansible_system == 'Darwin'` limits the install step to macOS.
- `Apply base manifests` deploys the app stack.
- `kubectl apply -k {{ project_dir }}/deployments/k8s/base` applies the Kustomize base layer.
- `Apply monitoring manifests` deploys Prometheus.
- `kubectl apply -k {{ project_dir }}/deployments/k8s/monitoring` applies the monitoring layer.
- `Display services` prints the services so you can confirm exposure.
- `kubectl get svc -n krtms` lists the services in the namespace.

### 3.4 Kustomize Base

File: `deployments/k8s/base/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: krtms
resources:
  - namespace.yaml
  - nats.yaml
  - rbac-event-collector.yaml
  - event-collector.yaml
  - analyzer.yaml
  - alert-manager.yaml
```

Explanation:
- `apiVersion: kustomize.config.k8s.io/v1beta1` sets the Kustomize schema version.
- `kind: Kustomization` identifies this file as a Kustomize entry point.
- `namespace: krtms` ensures all resources in this overlay are created in the `krtms` namespace.
- `resources:` starts the list of included manifests.
- `namespace.yaml` creates the namespace itself.
- `nats.yaml` deploys the NATS message bus.
- `rbac-event-collector.yaml` grants the event collector permission to watch pods.
- `event-collector.yaml` deploys the event collector service and service object.
- `analyzer.yaml` deploys the analyzer service and service object.
- `alert-manager.yaml` deploys the alert manager service and service object.

### 3.5 Namespace Manifest

File: `deployments/k8s/base/namespace.yaml`

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: krtms
```

Explanation:
- `apiVersion: v1` uses the core Kubernetes API.
- `kind: Namespace` declares a namespace resource.
- `metadata:` starts metadata.
- `name: krtms` names the namespace used by the whole stack.

### 3.6 NATS Manifest

File: `deployments/k8s/base/nats.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nats
  namespace: krtms
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nats
  template:
    metadata:
      labels:
        app: nats
    spec:
      containers:
        - name: nats
          image: nats:2.10-alpine
          ports:
            - containerPort: 4222
---
apiVersion: v1
kind: Service
metadata:
  name: nats
  namespace: krtms
spec:
  selector:
    app: nats
  ports:
    - port: 4222
      targetPort: 4222
```

Explanation by line or block:
- The first document starts a Deployment.
- `apiVersion: apps/v1` uses the apps API group.
- `kind: Deployment` declares a stateless workload.
- `metadata.name: nats` names the deployment.
- `namespace: krtms` puts it in the stack namespace.
- `replicas: 1` runs a single NATS instance.
- `selector.matchLabels.app: nats` tells the deployment which pods it owns.
- `template.metadata.labels.app: nats` labels the pod template so the selector matches.
- `containers:` begins the pod container list.
- `name: nats` names the container.
- `image: nats:2.10-alpine` uses the official lightweight NATS image.
- `ports.containerPort: 4222` exposes the NATS client port.
- `---` starts the second YAML document.
- The Service exposes NATS inside the cluster.
- `selector.app: nats` routes traffic to the NATS pods.
- `port: 4222` exposes the service on the standard NATS port.
- `targetPort: 4222` forwards traffic to the container port.

### 3.7 Event Collector Manifest

File: `deployments/k8s/base/event-collector.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: event-collector
  namespace: krtms
spec:
  replicas: 1
  selector:
    matchLabels:
      app: event-collector
  template:
    metadata:
      labels:
        app: event-collector
    spec:
      serviceAccountName: event-collector-sa
      containers:
        - name: event-collector
          image: ghcr.io/your-org/event-collector:latest
          imagePullPolicy: IfNotPresent
          env:
            - name: NATS_URL
              value: nats://nats:4222
            - name: HTTP_ADDR
              value: :8080
          ports:
            - name: http
              containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: event-collector
  namespace: krtms
spec:
  selector:
    app: event-collector
  ports:
    - name: http
      port: 8080
      targetPort: 8080
```

Explanation by line or block:
- The Deployment runs the event collector service.
- `replicas: 1` starts one collector pod.
- `selector.matchLabels.app: event-collector` identifies the pod managed by the deployment.
- `template.metadata.labels.app: event-collector` labels the pod template to match the selector.
- `serviceAccountName: event-collector-sa` gives the pod its dedicated RBAC identity.
- `name: event-collector` names the container.
- `image: ghcr.io/your-org/event-collector:latest` points to the container image.
- `imagePullPolicy: IfNotPresent` avoids pulling if the image already exists locally.
- `NATS_URL` tells the service how to reach NATS.
- `HTTP_ADDR` sets the health and metrics listen address.
- `containerPort: 8080` exposes the HTTP port.
- The Service exposes the collector to other cluster components.
- `selector.app: event-collector` routes traffic to the collector pod.
- `port: 8080` exposes the service port.
- `targetPort: 8080` forwards to the pod HTTP port.

### 3.8 Analyzer Manifest

File: `deployments/k8s/base/analyzer.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: analyzer
  namespace: krtms
spec:
  replicas: 1
  selector:
    matchLabels:
      app: analyzer
  template:
    metadata:
      labels:
        app: analyzer
    spec:
      containers:
        - name: analyzer
          image: ghcr.io/your-org/analyzer:latest
          imagePullPolicy: IfNotPresent
          env:
            - name: NATS_URL
              value: nats://nats:4222
            - name: HTTP_ADDR
              value: :8081
          ports:
            - name: http
              containerPort: 8081
---
apiVersion: v1
kind: Service
metadata:
  name: analyzer
  namespace: krtms
spec:
  selector:
    app: analyzer
  ports:
    - name: http
      port: 8081
      targetPort: 8081
```

Explanation by line or block:
- The Deployment runs the threat-detection logic.
- `replicas: 1` starts one analyzer pod.
- The selector and pod labels ensure ownership and routing.
- `image: ghcr.io/your-org/analyzer:latest` sets the analyzer image.
- `NATS_URL` tells the analyzer where to subscribe and publish.
- `HTTP_ADDR` exposes health, metrics, and the Falco endpoint.
- `containerPort: 8081` exposes the HTTP API port.
- The Service provides stable cluster access to the analyzer.
- `port: 8081` exposes the analyzer service.
- `targetPort: 8081` forwards to the pod port.

### 3.9 Alert Manager Manifest

File: `deployments/k8s/base/alert-manager.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alert-manager
  namespace: krtms
spec:
  replicas: 1
  selector:
    matchLabels:
      app: alert-manager
  template:
    metadata:
      labels:
        app: alert-manager
    spec:
      containers:
        - name: alert-manager
          image: ghcr.io/your-org/alert-manager:latest
          imagePullPolicy: IfNotPresent
          env:
            - name: NATS_URL
              value: nats://nats:4222
            - name: HTTP_ADDR
              value: :8082
            # Configure one or both channels:
            # - SLACK_WEBHOOK_URL
            # - SMTP_HOST / SMTP_PORT / SMTP_USER / SMTP_PASS / SMTP_TO
          ports:
            - name: http
              containerPort: 8082
---
apiVersion: v1
kind: Service
metadata:
  name: alert-manager
  namespace: krtms
spec:
  selector:
    app: alert-manager
  ports:
    - name: http
      port: 8082
      targetPort: 8082
```

Explanation by line or block:
- The Deployment runs the notification and dashboard service.
- `replicas: 1` starts one alert-manager pod.
- The selector and labels connect the deployment to its pod.
- `image: ghcr.io/your-org/alert-manager:latest` sets the alert-manager image.
- `NATS_URL` connects the service to the threat alert stream.
- `HTTP_ADDR` exposes the dashboard and API.
- The comment block lists optional notification settings.
- `containerPort: 8082` exposes the UI and API port.
- The Service makes the frontend and API reachable inside the cluster.
- `port: 8082` exposes the service port.
- `targetPort: 8082` forwards to the container port.

### 3.10 RBAC for Event Collector

File: `deployments/k8s/base/rbac-event-collector.yaml`

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: event-collector-sa
  namespace: krtms
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: event-collector-pod-reader
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: event-collector-pod-reader-binding
subjects:
  - kind: ServiceAccount
    name: event-collector-sa
    namespace: krtms
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: event-collector-pod-reader
```

Explanation by line or block:
- `ServiceAccount` creates the identity used by the event collector pod.
- `event-collector-sa` is the named service account.
- `ClusterRole` defines cluster-wide permissions.
- `rules:` begins the permission list.
- `apiGroups: [""]` means the core Kubernetes API group.
- `resources: ["pods"]` limits access to pod objects.
- `verbs: ["get", "list", "watch"]` allows reading and watching pods.
- `ClusterRoleBinding` connects the role to the service account.
- `subjects:` identifies who gets the role.
- `kind: ServiceAccount` means the subject is a service account.
- `roleRef:` points to the ClusterRole to bind.

### 3.11 Monitoring Kustomize Base

File: `deployments/k8s/monitoring/kustomization.yaml`

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: krtms
resources:
  - prometheus.yaml
```

Explanation:
- `apiVersion` and `kind` define this as a Kustomize config.
- `namespace: krtms` keeps monitoring in the same namespace.
- `prometheus.yaml` deploys Prometheus for metrics scraping.

### 3.12 Prometheus Manifest

File: `deployments/k8s/monitoring/prometheus.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: krtms
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
      - job_name: event-collector
        static_configs:
          - targets: ["event-collector:8080"]
      - job_name: analyzer
        static_configs:
          - targets: ["analyzer:8081"]
      - job_name: alert-manager
        static_configs:
          - targets: ["alert-manager:8082"]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: krtms
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
        - name: prometheus
          image: prom/prometheus:v2.54.1
          args:
            - --config.file=/etc/prometheus/prometheus.yml
          ports:
            - containerPort: 9090
          volumeMounts:
            - name: config
              mountPath: /etc/prometheus
      volumes:
        - name: config
          configMap:
            name: prometheus-config
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: krtms
spec:
  selector:
    app: prometheus
  ports:
    - port: 9090
      targetPort: 9090
```

Explanation by line or block:
- The ConfigMap stores the Prometheus configuration file.
- `prometheus.yml: |` embeds the config text.
- `global.scrape_interval: 15s` tells Prometheus how often to scrape.
- `scrape_configs:` defines scrape targets.
- `event-collector`, `analyzer`, and `alert-manager` each get a job.
- Each `targets` entry points to the service name and port inside the cluster.
- The Deployment runs Prometheus.
- `image: prom/prometheus:v2.54.1` uses the official Prometheus image.
- `--config.file=/etc/prometheus/prometheus.yml` tells Prometheus to read the mounted config.
- `containerPort: 9090` exposes the Prometheus web UI.
- `volumeMounts:` and `volumes:` mount the ConfigMap into the pod.
- The Service exposes Prometheus inside the cluster.
- `port: 9090` publishes the service port.
- `targetPort: 9090` forwards to the container port.

### 3.13 Shell Deploy Script

File: `scripts/deploy.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

kubectl apply -k deployments/k8s/base
kubectl apply -k deployments/k8s/monitoring
kubectl get pods -n krtms
```

Explanation:
- `#!/usr/bin/env bash` selects Bash as the shell.
- `set -euo pipefail` makes the script fail fast and safely.
- `kubectl apply -k deployments/k8s/base` deploys the application layer.
- `kubectl apply -k deployments/k8s/monitoring` deploys Prometheus.
- `kubectl get pods -n krtms` confirms the result after deployment.

## 4. Practical Summary

If you want the short version of the whole stack:
- Jenkins builds, scans, pushes, deploys, and optionally smoke-tests.
- Ansible provides a local setup-and-deploy path.
- Kustomize defines the Kubernetes resources.
- NATS carries events between services.
- Prometheus collects metrics.
- Alert Manager UI shows the latest alerts and summary cards.

## 5. Recommended Use

- Use `make smoke` or the Jenkins smoke stage to validate the alert pipeline.
- Use `kubectl port-forward -n krtms svc/alert-manager 8082:8082` to open the frontend.
- Use `kubectl port-forward -n krtms svc/prometheus 9090:9090` to inspect raw metrics.
- Use `ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/setup_and_deploy.yml` for local automation.

## 6. Notes

The deployment flow is currently optimized for Minikube or another cluster with `kubectl` access from the machine running Jenkins or Ansible.

If you want, this document can be split into three separate docs later:
- one for Kubernetes deployment,
- one for Ansible,
- one for Jenkins CI/CD.
