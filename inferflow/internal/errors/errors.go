package errors

type ParsingError struct {
	ErrorMsg string
}

func (m *ParsingError) Error() string {
	return m.ErrorMsg
}

type BadRequestError struct {
	ErrorMsg string
}

func (m *BadRequestError) Error() string {
	return m.ErrorMsg
}

type RequestError struct {
	ErrorMsg string
}

func (m *RequestError) Error() string {
	return m.ErrorMsg
}
