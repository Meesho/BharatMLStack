package inferflow

func validateInferflowRequest(m *InferflowRequest) bool {

	if len(*m.Entity) == 0 ||
		len(*m.EntityIds) == 0 ||
		m.ModelConfigId == "" ||
		len(*m.Entity) != len(*m.EntityIds) {
		return false
	}
	return true
}
