package errors

type BadRequestError struct {
	ErrorMsg string
}

func (m *BadRequestError) Error() string {
	return m.ErrorMsg
}

type ComponentError struct {
	ErrorMsg string
}

func (m *ComponentError) Error() string {
	return m.ErrorMsg
}

type InvalidComponentError struct {
	ErrorMsg string
}

func (m *InvalidComponentError) Error() string {
	return m.ErrorMsg
}

type InvalidDagError struct {
	ErrorMsg string
}

func (m *InvalidDagError) Error() string {
	return m.ErrorMsg
}
