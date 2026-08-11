// SPDX-License-Identifier: MIT

package spaceship

import "testing"

func TestRecordName(t *testing.T) {
	tests := []struct {
		name string
		zone string
		fqdn string
		want string
	}{
		{
			name: "simple apex challenge",
			fqdn: "_acme-challenge.example.com.",
			zone: "example.com.",
			want: "_acme-challenge",
		},
		{
			name: "subdomain challenge",
			fqdn: "_acme-challenge.sub.example.com.",
			zone: "example.com.",
			want: "_acme-challenge.sub",
		},
		{
			name: "empty zone returns trimmed fqdn",
			fqdn: "_acme-challenge.example.com.",
			zone: "",
			want: "_acme-challenge.example.com",
		},
		{
			name: "zone without trailing dot still works",
			fqdn: "_acme-challenge.example.com.",
			zone: "example.com",
			want: "_acme-challenge",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recordName(tc.zone, tc.fqdn)
			if got != tc.want {
				t.Fatalf("recordName(%q, %q) = %q, want %q", tc.zone, tc.fqdn, got, tc.want)
			}
		})
	}
}
