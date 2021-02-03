FROM golang:1.15-buster as build
ARG ARCH=amd64
ENV CGO_ENABLED=0
WORKDIR /work
RUN mkdir -p /etc/ct-monitor /var/log/ct-monitor \
    && chown nobody:nogroup /etc/ct-monitor /var/log/ct-monitor
COPY . .
RUN make ARCH=${ARCH}

FROM scratch
ARG ARCH=amd64
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build --chown=nobody:nogroup /etc/ct-monitor /etc/ct-monitor
COPY --from=build --chown=nobody:nogroup /var/log/ct-monitor /var/log/ct-monitor
COPY --from=build /tmp/ct-monitor/artifacts/ct-monitor-linux-${ARCH} /ct-monitor

USER nobody:nogroup

ENTRYPOINT [ "/ct-monitor" ]
