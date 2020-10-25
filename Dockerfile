FROM quay.io/cybozu/ubuntu:20.04 as certs

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY artifacts/ct-monitor-linux-amd64 /ct-monitor
ENTRYPOINT [ "/ct-monitor" ]
