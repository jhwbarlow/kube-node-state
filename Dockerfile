FROM golang:1.15 AS builder
COPY . /tmp/src/
RUN cd /tmp/src/cmd && \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o /tmp/bin/kube-node-state && \
    chmod 500 /tmp/bin/kube-node-state

FROM busybox AS env-builder
RUN truncate -s0 /etc/passwd /etc/group && \
    adduser -h / -g 'Kube Node State' -s /bin/nologin -D -H -u 10034 kube-node-state

FROM scratch
COPY --from=env-builder /etc/passwd /etc/group /etc/
USER kube-node-state:kube-node-state
COPY --from=builder --chown=kube-node-state:kube-node-state /tmp/bin/kube-node-state /usr/bin/kube-node-state
ENTRYPOINT [ "/usr/bin/kube-node-state" ]
