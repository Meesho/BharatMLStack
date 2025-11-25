package skye

func GetSkyeClient(version int) SkyeClient {
	switch version {
	case 1:
		return InitV1Client()
	default:
		return nil
	}
}
