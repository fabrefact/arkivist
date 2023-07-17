pipeline {
  agent any
  stages {
    stage('Build') {
      steps {
        sh 'go build'
      }
    }
    stage('Test') {
      steps {
        sh 'go install github.com/jstemmer/go-junit-report'
        sh 'go test -covermode=atomic -coverprofile=coverage.out'
        sh 'go env'
        sh 'go-junit-report'
      }
    }
  }
  // post {
  //  always {
  //    junit 'report.xml'
  //  }
  //}
}