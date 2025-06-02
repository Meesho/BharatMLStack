package listeners

type FeatureConsumer interface {
	Init()
	Consume()
}
