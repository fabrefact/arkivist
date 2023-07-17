pipeline {
  agent any
  stages {
    stage('Build') {
      steps {
        // Bare bones basic build-the-code step
        sh 'go build'
      }
    }
    stage('Test') {
      steps {
        // Had to do this as a multi-line script to set the PATH correctly
        // Set the path, install the junit-report converter, then run your tests & pipe it to the converter
        sh '''
          export PATH=$PATH:$(go env GOPATH)/bin
          go install github.com/jstemmer/go-junit-report
          go test -v 2>&1 ./... | go-junit-report -set-exit-code > report.xml
        '''
      }
    }
  }
  post {
    // 'always' refers to doing this regardless of build status, this can also be set to success/failure/abort
    always {
      // Archive & parse the junit reports stored in the listed filename
      junit 'report.xml'
    }
  }
}
