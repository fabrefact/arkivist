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
        sh '''
          export PATH=$PATH:$(go env GOPATH)/bin
          go install github.com/jstemmer/go-junit-report
          go test -v 2>&1 ./... | go-junit-report -set-exit-code > report.xml
        '''
      }
    }
  }
  post {
    always {
      junit 'report.xml'
    }
  }
}