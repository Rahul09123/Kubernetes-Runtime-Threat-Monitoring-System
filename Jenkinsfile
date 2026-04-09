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
