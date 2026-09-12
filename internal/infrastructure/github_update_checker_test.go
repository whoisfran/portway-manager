package infrastructure

import "testing"

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"parche mas nuevo", "v0.4.0", "v0.4.1", true},
		{"minor mas nuevo", "v0.4.9", "v0.5.0", true},
		{"major mas nuevo", "v0.9.0", "v1.0.0", true},
		{"misma version", "v0.4.0", "v0.4.0", false},
		{"version actual ya es mas nueva", "v0.5.0", "v0.4.0", false},
		{"sin el prefijo v", "0.4.0", "0.4.1", true},
		{"sufijo de prerelease se ignora al parsear", "v0.4.0", "v0.4.1-rc1", true},
		// "dev" (build local fuera de un release) no se puede
		// interpretar como semver: nunca debe ofrecer una
		// "actualizacion" en ese caso, para no arriesgar un falso
		// positivo confuso.
		{"version actual dev", "dev", "v1.0.0", false},
		{"comparacion numerica, no lexicografica (v0.10.0 > v0.9.0)", "v0.9.0", "v0.10.0", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isNewerVersion(c.current, c.latest)
			if got != c.want {
				t.Errorf("isNewerVersion(%q, %q) = %v, quiero %v", c.current, c.latest, got, c.want)
			}
		})
	}
}
