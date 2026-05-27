package server

import "testing"

func TestNormalizeTDengineRESTURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://10.40.81.130:6041", "http://10.40.81.130:6041"},
		{"http://10.40.81.130:6060/rest/sql", "http://10.40.81.130:6060"},
		{"http://10.40.81.130:6060/rest/sql/", "http://10.40.81.130:6060"},
	}
	for _, c := range cases {
		if got := NormalizeTDengineRESTURL(c.in); got != c.want {
			t.Fatalf("NormalizeTDengineRESTURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
