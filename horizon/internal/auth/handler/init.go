package handler

func NewAuthenticator(version int) Authenticator {
	switch version {
	case 1:
		return InitAuthHandler()
	default:
		return nil
	}
}
