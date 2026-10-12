package flash

type Type string

type Message struct {
	Type    Type   `json:"type"`
	Message string `json:"message"`
}

type Messages []Message

const (
	TypeSuccess = Type("success")
	TypeError   = Type("error")
	TypeWarning = Type("warning")
	TypeInfo    = Type("info")
)

func (m Message) IsSuccess() bool {
	return m.Type == TypeSuccess
}

func (m Message) IsError() bool {
	return m.Type == TypeError
}

func (m Message) IsWarning() bool {
	return m.Type == TypeWarning
}

func (m Message) IsInfo() bool {
	return m.Type == TypeInfo || len(m.Type) == 0
}

func WithSuccess(msg string) Message {
	return Message{
		Type:    TypeSuccess,
		Message: msg,
	}
}

func WithError(msg string) Message {
	return Message{
		Type:    TypeError,
		Message: msg,
	}
}

func WithInfo(msg string) Message {
	return Message{
		Type:    TypeInfo,
		Message: msg,
	}
}

func WithWarning(msg string) Message {
	return Message{
		Type:    TypeWarning,
		Message: msg,
	}
}
