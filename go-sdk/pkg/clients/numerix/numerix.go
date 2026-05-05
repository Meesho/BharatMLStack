package numerix

type NumerixClient interface {
	RetrieveScore(req *NumerixRequest) (*NumerixResponse, error)
}
