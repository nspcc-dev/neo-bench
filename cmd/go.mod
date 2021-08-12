module github.com/nspcc-dev/neo-bench

go 1.14

require (
	github.com/Workiva/go-datastructures v1.0.53
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/docker/docker v20.10.8+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/fatih/color v1.12.0
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/k14s/ytt v0.30.0
	github.com/mailru/easyjson v0.7.1
	github.com/moby/moby v20.10.8+incompatible
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nspcc-dev/neo-go v0.97.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.1
	github.com/valyala/fasthttp v1.9.0
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.18.1
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/pkg/errors v0.8.1 => github.com/pkg/errors v0.9.1 // see https://github.com/containerd/containerd/issues/4703#issuecomment-736542317
