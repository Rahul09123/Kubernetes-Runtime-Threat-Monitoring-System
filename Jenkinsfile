pipeline {
  agent any

  environment {
    REGISTRY = 'ghcr.io/your-org'
    TAG = "${env.BUILD_NUMBER}"
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
        sh '''docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:0.53.2 image --severity HIGH,CRITICAL --exit-code 1 $REGISTRY/event-collector:$TAG'''
        sh '''docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:0.53.2 image --severity HIGH,CRITICAL --exit-code 1 $REGISTRY/analyzer:$TAG'''
        sh '''docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:0.53.2 image --severity HIGH,CRITICAL --exit-code 1 $REGISTRY/alert-manager:$TAG'''
      }
    }

    stage('Push Images') {
      when {
        expression { return env.BRANCH_NAME == 'main' }
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
  }
}
