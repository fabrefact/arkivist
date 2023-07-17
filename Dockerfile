# Specify amd64, since arm64 is the default on our macs and
# it gets weird from a go binary perspective real quick
FROM --platform=linux/amd64 jenkins/jenkins:lts

# Change to the root user to be able to install golang properly
USER root

# Install golang 1.20.6 & tidy up
RUN curl -Ls https://go.dev/dl/go1.20.6.linux-amd64.tar.gz -o 'go1.20.6.linux-amd64.tar.gz' \
    && tar -C /usr/local -xzf go1.20.6.linux-amd64.tar.gz \
    && rm -rf go1.20.6.linux-amd64.tar.gz

# Change back to the jenkins user
USER jenkins

# Add golang to our PATH
ENV PATH="$PATH:/usr/local/go/bin"

# Set the entrypoint to be the same as parent container so everything works the same
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/jenkins.sh"]
