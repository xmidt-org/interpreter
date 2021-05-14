package validation

type testError struct {
	err error
	tag Tag
}

func (t testError) Error() string {
	return t.err.Error()
}

func (t testError) Tag() Tag {
	return t.tag
}

func (t testError) Unwrap() error {
	return t.err
}
