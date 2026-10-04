package sysinfo

import (
	"context"
	"errors"
	"testing"

	"github.com/josepbrotons/mobile-dev-tools/internal/runner"
)

func TestCheck(t *testing.T) {
	df := "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/disk3s1 971350180 500000000 52428800 91% /\n"
	tests := []struct {
		name string
		fake *runner.Fake
		want Info
	}{
		{
			name: "apple silicon",
			fake: &runner.Fake{Output: map[string]string{
				"sw_vers": "15.1\n", "uname": "arm64\n", "sysctl": "0\n", "df": df,
			}},
			want: Info{MacOS: "15.1", Arch: "arm64", FreeGB: 50},
		},
		{
			name: "rosetta",
			fake: &runner.Fake{Output: map[string]string{"uname": "x86_64\n", "sysctl": "1\n"}},
			want: Info{Arch: "x86_64", Rosetta: true},
		},
		{
			name: "all probes fail",
			fake: &runner.Fake{Fail: map[string]error{"": errors.New("boom")}},
			want: Info{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Check(context.Background(), tt.fake); got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
