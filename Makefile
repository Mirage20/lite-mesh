


.PHONY: envoy-bootstrap
envoy-bootstrap:
	go build  -o ./out/envoy-bootstrap/envoy-bootstrap ./cmd/envoy-bootstrap/
	cp ./docker/envoy-bootstrap/Dockerfile ./out/envoy-bootstrap/
	cp envoy-bootstrap-template.yaml ./out/envoy-bootstrap/
	docker build ./out/envoy-bootstrap/ -t mirage20/envoy-proxy
