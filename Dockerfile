FROM scratch
LABEL org.opencontainers.image.authors="Hsn723" \
      org.opencontainers.image.title="ct-monitor" \
      org.opencontainers.image.source="https://github.com/hsn723/ct-monitor"
COPY LICENSE /LICENSE
COPY ct-monitor /

USER 65534:65534

ENTRYPOINT [ "/ct-monitor" ]
