module github.com/TriggerMail/lazylru/bench

go 1.16

replace github.com/TriggerMail/lazylru => ../

require (
	github.com/TriggerMail/lazylru v0.0.0-00010101000000-000000000000
	github.com/hashicorp/golang-lru v0.5.4
	go.uber.org/zap v1.16.0
)
