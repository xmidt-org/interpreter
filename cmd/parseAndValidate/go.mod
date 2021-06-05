module github.com/xmidt-org/interpreter/cmd/parseAndValidate

go 1.15

require (
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/xmidt-org/arrange v0.3.0
	github.com/xmidt-org/bascule v0.9.0
	github.com/xmidt-org/httpaux v0.2.1
	github.com/xmidt-org/interpreter v0.0.4-0.20210527001806-f935834c14ff
	go.uber.org/fx v1.13.1
)

replace github.com/xmidt-org/interpreter v0.0.4-0.20210527001806-f935834c14ff => github.com/xmidt-org/interpreter v0.0.4-0.20210604185111-7d1867efaa8c