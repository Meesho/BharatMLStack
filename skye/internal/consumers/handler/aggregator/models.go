package aggregator

type Payload struct {
	Entity      string
	Model       string
	CandidateId string
	Columns     map[string]string
}

type Response struct {
	CompleteData map[string]interface{}
	DeltaData    map[string]interface{}
}
