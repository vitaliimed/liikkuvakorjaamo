package registration

import "testing"

func TestRegistrationPolicy(t *testing.T) {
	for input, want := range map[string]string{"abc123": "ABC-123", " ABC-123 ": "ABC-123", "abc 123": "ABC-123", "123abc": "123-ABC", "123 abc": "123-ABC", "123 - abc": "123-ABC", "000xyz": "000-XYZ"} {
		if got := Normalize(input); got != want || !Valid(input) {
			t.Errorf("%q: got %q, want %q", input, got, want)
		}
		if Key(input) != want[:3]+want[4:] {
			t.Errorf("key changed for %q", input)
		}
	}
	for _, input := range []string{"", "ABC-1", "ABC-12", "AB-123", "123-AB", "ABCD123", "ABC1234", "1ABC23", "АВС123", "ABC/123", "ABC--123", "ABC-123!", "ABC\n123"} {
		if Valid(input) {
			t.Errorf("invalid input accepted: %q -> %q", input, Normalize(input))
		}
	}
}
