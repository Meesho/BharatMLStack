package inferflow

func GetInferflowClient(version int) InferflowClient {
	switch version {
	case 1:
		return InitV1Client()
	default:
		return nil
	}
}

func GetInferflowClientFromConfig(version int, conf ClientConfig, callerId string) InferflowClient {
	switch version {
	case 1:
		return InitV1ClientFromConfig(conf, callerId)
	default:
		return nil
	}
}
