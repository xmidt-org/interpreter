package validation

type testError struct {
	err  error
	tag  Tag
	tags []Tag
}

func (t testError) Error() string {
	return t.err.Error()
}

func (t testError) Tag() Tag {
	if t.tag == Unknown && len(t.tags) > 0 {
		return MultipleTags
	}
	return t.tag
}

func (t testError) Tags() []Tag {
	return t.tags
}

func (t testError) Unwrap() error {
	return t.err
}
