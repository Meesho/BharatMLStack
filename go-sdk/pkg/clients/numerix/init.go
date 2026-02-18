package numerix

func GetNumerixClient(version int, configBytes []byte) NumerixClient {
	switch version {
	case 1:
		return InitV1Client(configBytes)
	default:
		return nil
	}
}
