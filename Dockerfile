FROM --platform=linux/amd64 jenkins/jenkins:lts

USER root

# Install golang 1.20.6 & tidy up
RUN curl -Ls https://go.dev/dl/go1.20.6.linux-amd64.tar.gz -o 'go1.20.6.linux-amd64.tar.gz' \
    && tar -C /usr/local -xzf go1.20.6.linux-amd64.tar.gz \
    && rm -rf go1.20.6.linux-amd64.tar.gz

USER jenkins

ENV PATH="$PATH:/usr/local/go/bin"

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/jenkins.sh"]
