ARG BASE_PREFIX=mirror.gcr.io/library/
FROM gcr.io/gcp-runtimes/ubuntu_16_0_4

ENV KUBECTL_VERSION v1.12.8
ENV KUBECTL_URL https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
RUN curl -O "${KUBECTL_URL}"
RUN chmod +x kubectl
RUN mv kubectl /usr/bin/kubectl

COPY ./devloop /usr/bin/devloop
RUN chmod +x /usr/bin/devloop

COPY ./test-build /test-build
COPY ./devloop.yaml ./devloop.yaml
