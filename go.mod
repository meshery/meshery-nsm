module github.com/layer5io/meshery-nsm

go 1.15

replace github.com/kudobuilder/kuttl => github.com/layer5io/kuttl v0.4.1-0.20200806180306-b7e46afd657f

require (
	github.com/containerd/containerd v1.3.4 // indirect
	github.com/layer5io/meshery-adapter-library v0.1.20
	github.com/layer5io/meshkit v0.2.17
	github.com/layer5io/service-mesh-performance v0.3.3
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.11.0 // indirect
	k8s.io/api v0.18.12 // indirect
	k8s.io/apimachinery v0.18.12 // indirect
	k8s.io/client-go v0.18.12 // indirect
)
