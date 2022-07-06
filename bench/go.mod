module github.com/TriggerMail/lazylru/bench

go 1.18

replace github.com/TriggerMail/lazylru => ../

replace github.com/TriggerMail/lazylru/generic => ../generic

require (
	github.com/TriggerMail/lazylru v0.2.0
	github.com/TriggerMail/lazylru/generic v0.2.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/stretchr/testify v1.8.0
	go.uber.org/zap v1.21.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
