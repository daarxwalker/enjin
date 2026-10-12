package validate_test

import (
	"testing"

	"github.com/daarxwalker/enjin/validate"

	"github.com/stretchr/testify/require"
)

func TestVerdicts(t *testing.T) {
	t.Run(
		"text", func(t *testing.T) {
			tests := []struct {
				name    string
				value   string
				allowed []string
				want    []validate.Verdict
			}{
				{name: "empty", value: "", want: []validate.Verdict{validate.Required}},
				{name: "diacritics count as runes", value: "žluťák", want: nil},
				{name: "too short", value: "ab", want: []validate.Verdict{validate.MinLen(3)}},
				{name: "too long with diacritics", value: "příliš žluťoučký", want: []validate.Verdict{validate.MaxLen(6)}},
				{name: "allowed", value: "abc", allowed: []string{"abc", "def"}, want: nil},
				{name: "not allowed", value: "xyz", allowed: []string{"abc", "def"}, want: []validate.Verdict{validate.Invalid("abc", "def")}},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						require.Equal(t, tt.want, validate.Text(tt.value, 3, 6, tt.allowed...))
					},
				)
			}
		},
	)

	t.Run(
		"numbers", func(t *testing.T) {
			require.Equal(t, []validate.Verdict{validate.Required}, validate.Int(0, 1, 10))
			require.Nil(t, validate.Int(5, 1, 10))
			require.Equal(t, []validate.Verdict{validate.Min(1)}, validate.Int(-1, 1, 10))
			require.Equal(t, []validate.Verdict{validate.Max(10)}, validate.Int(11, 1, 10))
			require.Equal(t, []validate.Verdict{validate.Required}, validate.Float64(0, 1, 10))
			require.Nil(t, validate.Float64(9.5, 1, 10))
			require.Equal(t, []validate.Verdict{validate.Min(1)}, validate.Float64(0.5, 1, 10))
			require.Equal(t, []validate.Verdict{validate.Max(10)}, validate.Float64(10.1, 1, 10))
		},
	)

	t.Run(
		"special", func(t *testing.T) {
			tests := []struct {
				value  string
				email  bool
				uuidv7 bool
			}{
				{value: "", email: false, uuidv7: false},
				{value: "user@example.com", email: true},
				{value: "user@example", email: false},
				{value: "us er@example.com", email: false},
				{value: "0190f2a8-7a3e-7c3b-9d4e-1f2a3b4c5d6e", uuidv7: true},
				{value: "0190f2a8-7a3e-4c3b-9d4e-1f2a3b4c5d6e", uuidv7: false},
				{value: "0190f2a8-7a3e-7c3b-1d4e-1f2a3b4c5d6e", uuidv7: false},
				{value: "0190f2a8-7a3e-7c3b-9d4e-1f2a3b4c5d6e0", uuidv7: false},
			}
			for _, tt := range tests {
				t.Run(
					tt.value, func(t *testing.T) {
						require.Equal(t, tt.email, validate.IsEmail(tt.value))
						require.Equal(t, tt.uuidv7, validate.IsUUIDv7(tt.value))
					},
				)
			}
			require.Nil(t, validate.Email(""))
			require.Equal(t, []validate.Verdict{validate.Invalid("email")}, validate.Email("nope"))
			require.Nil(t, validate.UUIDv7(""))
			require.Equal(t, []validate.Verdict{validate.Invalid("uuidv7")}, validate.UUIDv7("nope"))
		},
	)

	t.Run(
		"compose", func(t *testing.T) {
			require.Equal(
				t,
				[]validate.Verdict{validate.MinLen(3), validate.Invalid("email")},
				validate.Compose(validate.Text("a@", 3, 10), validate.Email("a@")),
			)
			require.Empty(t, validate.Compose())
		},
	)
}

func TestResult(t *testing.T) {
	t.Run(
		"collect verdicts", func(t *testing.T) {
			r := make(validate.Result)
			require.True(t, r.Zero())
			r.Text("name", "Jo", 3, 10)
			r.Email("email", "user@example.com")
			r.Int("age", 0, 18, 99)
			r.Add("name", validate.Invalid("custom"))
			require.False(t, r.Zero())
			require.True(t, r.Has("name"))
			require.False(t, r.Has("email"))
			require.Equal(t, []validate.Verdict{validate.MinLen(3), validate.Invalid("custom")}, r.Get("name"))
			require.Equal(t, []validate.Verdict{}, r.Get("email"))
			require.Equal(t, []validate.Verdict{validate.Required}, r.Get("age"))
		},
	)

	t.Run(
		"string is sorted by key", func(t *testing.T) {
			r := make(validate.Result)
			r.Add("zip", validate.Required)
			r.Add("age", validate.Min(18), validate.Max(99))
			r.Add("name", validate.Required)
			want := "age:min(18),max(99)\nname:required\nzip:required\n"
			for range 20 {
				require.Equal(t, want, r.String())
			}
			require.EqualError(t, r.Error(), want)
			require.Empty(t, make(validate.Result).String())
		},
	)
}
