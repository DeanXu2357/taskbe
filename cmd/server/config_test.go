package main

import "testing"

func TestLoadConfig(t *testing.T) {
	cases := []struct {
		port     string
		wantAddr string
		wantErr  bool
	}{
		{"", ":8080", false},
		{"9090", ":9090", false},
		{"0", "", true},
		{"65536", "", true},
		{"abc", "", true},
		{"-1", "", true},
	}
	for _, tc := range cases {
		cfg, err := loadConfig(func(key string) string {
			if key == "PORT" {
				return tc.port
			}
			return ""
		})

		if (err != nil) != tc.wantErr {
			t.Errorf("PORT=%q: err = %v, wantErr %v", tc.port, err, tc.wantErr)
			continue
		}
		if cfg.Addr != tc.wantAddr {
			t.Errorf("PORT=%q: Addr = %q, want %q", tc.port, cfg.Addr, tc.wantAddr)
		}
	}
}
