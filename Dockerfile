FROM golang:1.26-bookworm AS builder

WORKDIR /app

# Go module proxy / sumdb for networks where the default endpoints are slow or blocked
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

RUN apt-get update && apt-get install -y --no-install-recommends build-essential git && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=1 GOOS=linux go build -buildvcs=false -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o ./CLIProxyAPI ./cmd/server/

FROM debian:bookworm

# python3/pip/cron are used by scripts/wakeup_space.py (HF Space keepalive)
RUN apt-get update && apt-get install -y --no-install-recommends tzdata ca-certificates python3 python3-pip cron && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /CLIProxyAPI/scripts

COPY --from=builder ./app/CLIProxyAPI /CLIProxyAPI/CLIProxyAPI

COPY scripts/wakeup_space.py /CLIProxyAPI/scripts/wakeup_space.py

RUN pip3 install --no-cache-dir --break-system-packages huggingface_hub

COPY config.example.yaml /CLIProxyAPI/config.example.yaml

# Keepalive cron job: wake the HF Space every 10 minutes
RUN echo '*/10 * * * * cd /CLIProxyAPI && python3 /CLIProxyAPI/scripts/wakeup_space.py >> /var/log/wakeup.log 2>&1' > /etc/cron.d/wakeup && chmod 0644 /etc/cron.d/wakeup

RUN chmod +x /CLIProxyAPI/scripts/wakeup_space.py

WORKDIR /CLIProxyAPI

EXPOSE 8317

ENV TZ=Asia/Shanghai

RUN cp /usr/share/zoneinfo/${TZ} /etc/localtime && echo "${TZ}" > /etc/timezone

# Start cron, run the keepalive once (non-fatal), then the proxy server
CMD cron && (python3 /CLIProxyAPI/scripts/wakeup_space.py || true) && ./CLIProxyAPI
