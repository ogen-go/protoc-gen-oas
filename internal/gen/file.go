package gen

import "google.golang.org/protobuf/compiler/protogen"

// NewFile returns File instance.
func NewFile(f *protogen.File) (*File, error) {
	ss, err := NewServices(f.Services)
	if err != nil {
		return nil, err
	}

	ms, err := NewMessages(f.Messages)
	if err != nil {
		return nil, err
	}

	return &File{
		Generate: f.Generate,
		Services: ss,
		Messages: ms,
	}, nil
}

// NewFiles returns Files instance.
func NewFiles(fs []*protogen.File) (Files, error) {
	files := make(Files, 0, len(fs))

	for _, f := range fs {
		file, err := NewFile(f)
		if err != nil {
			return nil, err
		}

		files = append(files, file)
	}

	return files, nil
}

// File instance.
type File struct {
	Generate bool
	Services Services
	Messages Messages
}

// Files is File slice instance.
type Files []*File
