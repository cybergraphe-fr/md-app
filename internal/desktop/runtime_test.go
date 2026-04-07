package desktop

import "testing"

func TestValidateRemoteAPIURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "https", in: "https://md.cybergraphe.fr", want: "https://md.cybergraphe.fr"},
		{name: "http", in: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "trim trailing slash", in: "https://md.cybergraphe.fr/", want: "https://md.cybergraphe.fr"},
		{name: "reject empty", in: "", wantErr: true},
		{name: "reject relative", in: "/api", wantErr: true},
		{name: "reject javascript", in: "javascript:alert(1)", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateRemoteAPIURL(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("validateRemoteAPIURL(%q)=%q want=%q", tc.in, got, tc.want)
			}
		})
	}
}

func TestStripSecureCookieFlag(t *testing.T) {
	in := "md-workspace=abc; Path=/; Max-Age=3600; HttpOnly; Secure; SameSite=Lax"
	got := stripSecureCookieFlag(in)
	want := "md-workspace=abc; Path=/; Max-Age=3600; HttpOnly; SameSite=Lax"
	if got != want {
		t.Fatalf("stripSecureCookieFlag()=%q want=%q", got, want)
	}
}
