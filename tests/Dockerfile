FROM golang:1.25.0-bookworm

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-venv postgresql-client curl tcpdump tshark default-jre-headless make \
    && rm -rf /var/lib/apt/lists/*
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:/opt/allure/bin:${PATH}"
COPY tests/requirements.txt /opt/requirements.txt
RUN pip install --no-cache-dir -r /opt/requirements.txt
ARG ALLURE_VERSION=2.34.1
RUN curl --fail --location --retry 3 \
    "https://github.com/allure-framework/allure2/releases/download/${ALLURE_VERSION}/allure-${ALLURE_VERSION}.tgz" \
    | tar xz -C /opt && mv /opt/allure-${ALLURE_VERSION} /opt/allure
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.26.0
COPY tests/runner.sh /opt/runner.sh
WORKDIR /workspace
ENTRYPOINT ["bash", "/opt/runner.sh"]
