FROM envoyproxy/envoy:v1.9.0

COPY envoy-bootstrap /usr/local/bin/envoy-bootstrap
COPY envoy-bootstrap-template.yaml /etc/conf/envoy-bootstrap-template.yaml

RUN useradd --uid 2020 envoy-proxy
RUN chown envoy-proxy /etc/conf

USER envoy-proxy


#ENTRYPOINT ["/usr/bin/dumb-init", "--"]
ENTRYPOINT ["/usr/local/bin/envoy-bootstrap"]
#CMD /usr/local/bin/envoy --v2-config-only -l $loglevel -c /etc/envoy/envoy.yaml


# docker run mirage20/envoy-proxy:latest --bootstrapTemplate="/etc/conf/envoy-bootstrap-template.yaml" --bootstrapConfig="/etc/conf/envoy-bootstrap.yaml" --envoyBinary="/usr/local/bin/envoy"
