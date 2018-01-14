package path

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	testcases := []struct {
		input    string
		expected Path
	}{
		{"otaru://vhost/fuga/hoge.txt", Path{"vhost", "fuga/hoge.txt"}},
		{"//vhost/fuga/hoge.txt", Path{"vhost", "fuga/hoge.txt"}},
		{"otaru:/fuga/hoge.txt", Path{"default", "fuga/hoge.txt"}},
		{"/1/2/3.txt", Path{"default", "1/2/3.txt"}},
	}

	for _, tc := range testcases {
		p, err := Parse(tc.input)
		if err != nil {
			t.Errorf("parse input: \"%s\" err: %v", tc.input, err)
		}
		if !reflect.DeepEqual(p, tc.expected) {
			t.Errorf("parse input: \"%s\" exp: %+v act: %+v", tc.input, tc.expected, p)
		}
	}
}
