package gen

import "google.golang.org/protobuf/compiler/protogen"

// NewMessage returns Message instance.
func NewMessage(m *protogen.Message) (*Message, error) {
	n := NewName(string(m.Desc.Name()))

	fs, err := NewFields(m.Fields)
	if err != nil {
		return nil, err
	}

	return &Message{
		Name:   n,
		Fields: fs,
	}, nil
}

// NewMessages returns Messages instance.
func NewMessages(ms []*protogen.Message) (Messages, error) {
	messages := make(Messages, 0, len(ms))

	for _, m := range ms {
		message, err := NewMessage(m)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}

// Message instance.
type Message struct {
	Name   Name
	Fields Fields
}

// Messages is Message slice instance.
type Messages []*Message
