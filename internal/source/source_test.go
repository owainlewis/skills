package source

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		explicit string
		wantURL  string
		wantPath string
		wantErr  bool
	}{
		{"shorthand", "owner/repo", "", "https://github.com/owner/repo.git", "", false},
		{"shorthand subpath", "owner/repo/skills/commit", "", "https://github.com/owner/repo.git", "skills/commit", false},
		{"shorthand host", "gitlab.com/owner/repo", "", "https://gitlab.com/owner/repo.git", "", false},
		{"shorthand host subpath", "gitlab.com/owner/repo/a/b", "", "https://gitlab.com/owner/repo.git", "a/b", false},
		{"https", "https://github.com/owner/repo", "", "https://github.com/owner/repo.git", "", false},
		{"https subpath", "https://github.com/owner/repo/skills/x", "", "https://github.com/owner/repo.git", "skills/x", false},
		{"https dotgit", "https://github.com/owner/repo.git", "", "https://github.com/owner/repo.git", "", false},
		{"ssh", "git@github.com:owner/repo.git", "", "git@github.com:owner/repo.git", "", false},
		{"local abs", "/tmp/my/repo", "", "/tmp/my/repo", "", false},
		{"local abs with path", "/tmp/my/repo", "skills/x", "/tmp/my/repo", "skills/x", false},
		{"local rel", "./repo", "", "./repo", "", false},
		{"file url", "file:///tmp/my/repo", "", "file:///tmp/my/repo", "", false},
		{"explicit path overrides", "owner/repo/ignored", "skills/real", "https://github.com/owner/repo.git", "skills/real", false},
		{"explicit path on ssh", "git@github.com:owner/repo.git", "skills/x", "git@github.com:owner/repo.git", "skills/x", false},
		{"empty", "", "", "", "", true},
		{"too short", "owner", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.raw, tt.explicit)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.GitURL != tt.wantURL {
				t.Errorf("GitURL = %q, want %q", got.GitURL, tt.wantURL)
			}
			if got.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", got.Path, tt.wantPath)
			}
			if got.Raw != tt.raw {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.raw)
			}
		})
	}
}
