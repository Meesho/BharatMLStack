package skye

func GetSkyeClient(version int) SkyeClient {
	switch version {
	case 1:
		return InitV1Client()
	default:
		return nil
	}
}

func GetSkyeClientFromConfig(version int, conf ClientConfig, callerId string) SkyeClient {
	switch version {
	case 1:
		return InitV1ClientFromConfig(conf, callerId)
	default:
		return nil
	}
}
