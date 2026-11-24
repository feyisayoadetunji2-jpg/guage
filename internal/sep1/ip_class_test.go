package sep1

import (
	"net"
	"testing"
)

func TestClassBoundaries(t *testing.T) {
	cases := []struct {
		address    string
		want       Class
		wantString string
		wantDial   bool
	}{
		{address: "64:ff9b::a00:1", want: ClassPrivate, wantString: "private", wantDial: false},
		{address: "64:ff9b::7f00:1", want: ClassLoopback, wantString: "loopback", wantDial: false},
		{address: "64:ff9b::808:808", want: ClassPublic, wantString: "public", wantDial: true},
		{address: "64:ff9b:0:1::a00:1", want: ClassPublic, wantString: "public", wantDial: true},
		{address: "2002:7f00:1::1", want: ClassLoopback, wantString: "loopback", wantDial: false},
		{address: "2002:e000:1::1", want: ClassMulticast, wantString: "multicast", wantDial: false},
		{address: "2002::", want: ClassUnspecified, wantString: "unspecified", wantDial: false},
		{address: "2002:0a00:1::1", want: ClassPrivate, wantString: "private", wantDial: false},
		{address: "::ffff:8.8.8.8", want: ClassPublic, wantString: "public", wantDial: true},
		{address: "::ffff:100.64.0.1", want: ClassCarrierNAT, wantString: "carrier-grade NAT", wantDial: false},
		{address: "100.64.0.1", want: ClassCarrierNAT, wantString: "carrier-grade NAT", wantDial: false},
		{address: "fc00::1", want: ClassUniqueLocal, wantString: "unique-local", wantDial: false},
	}
	for _, tc := range cases {
		t.Run(tc.address, func(t *testing.T) {
			ip := net.ParseIP(tc.address)
			if ip == nil {
				t.Fatalf("net.ParseIP(%q) = nil", tc.address)
			}
			got := ClassifyIP(ip)
			t.Logf("%s -> %s (dialable=%t)", tc.address, got, got.Dialable())
			if got != tc.want {
				t.Fatalf("ClassifyIP(%s) = %v, want %v", tc.address, got, tc.want)
			}
			if got.String() != tc.wantString {
				t.Fatalf("ClassifyIP(%s).String() = %q, want %q", tc.address, got.String(), tc.wantString)
			}
			if got.Dialable() != tc.wantDial {
				t.Fatalf("ClassifyIP(%s).Dialable() = %t, want %t", tc.address, got.Dialable(), tc.wantDial)
			}
		})
	}
}
