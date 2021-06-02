package validation

type testTaggedError struct {
	err error
	tag Tag
}

func (t testTaggedError) Error() string {
	return t.err.Error()
}

func (t testTaggedError) Tag() Tag {
	return t.tag
}

type testTaggedErrors struct {
	err  error
	tag  Tag
	tags []Tag
}

func (t testTaggedErrors) Error() string {
	return t.err.Error()
}

func (t testTaggedErrors) Tag() Tag {
	if t.tag == Unknown && len(t.tags) > 0 {
		return MultipleTags
	}
	return t.tag
}

func (t testTaggedErrors) Tags() []Tag {
	return t.tags
}

func (t testTaggedErrors) UniqueTags() []Tag {
	var tags []Tag
	existingTags := make(map[Tag]bool)

	for _, tag := range t.tags {
		if !existingTags[tag] {
			existingTags[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}

func (t testTaggedErrors) Unwrap() error {
	return t.err
}
