FROM envoyproxy/envoy:v1.9.0

RUN useradd --uid 2020 envoy-proxy

USER envoy-proxy

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD /usr/local/bin/envoy --v2-config-only -l $loglevel -c /etc/envoy/envoy.yaml
