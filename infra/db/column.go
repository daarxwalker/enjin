package db

type Column[T any] struct {
	Get  func(T) any
	Name string
	Cast string
}

func NewColumn[T any](name string, get func(T) any) Column[T] {
	return Column[T]{Name: name, Get: get}
}

func NewCast[T any](name, cast string, get func(T) any) Column[T] {
	return Column[T]{Name: name, Cast: cast, Get: get}
}

func ColumnText[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "text", get)
}

func ColumnBytea[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "bytea", get)
}

func ColumnInt[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "int", get)
}

func ColumnSmallInt[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "smallint", get)
}

func ColumnJSONB[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "jsonb", get)
}

func ColumnBool[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "boolean", get)
}

func ColumnTextArray[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "text[]", get)
}

func ColumnUUID[T any](name string, get func(T) any) Column[T] {
	return NewCast[T](name, "uuid", get)
}
