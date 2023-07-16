FROM jenkins/jenkins:lts

USER root

RUN apt-get update && apt-get install golang-go -y && apt-get clean

USER jenkins

ENTRYPOINT ["/usr/bin/tini" "--" "/usr/local/bin/jenkins.sh"]
