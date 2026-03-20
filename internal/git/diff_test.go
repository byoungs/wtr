package git

import (
	"testing"
)

const testDiff = `diff --git a/internal/auth/middleware.go b/internal/auth/middleware.go
index abc1234..def5678 100644
--- a/internal/auth/middleware.go
+++ b/internal/auth/middleware.go
@@ -42,7 +42,12 @@ func AuthMiddleware(next http.Handler) http.Handler {
 	mux := http.NewServeMux()
-	token := r.Header.Get("Authorization")
+	token, err := extractBearerToken(r)
+	if err != nil {
+		http.Error(w, "unauthorized", 401)
+		return
+	}
 	// validate token
diff --git a/internal/auth/token.go b/internal/auth/token.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/internal/auth/token.go
@@ -0,0 +1,5 @@
+package auth
+
+func extractBearerToken(r *http.Request) (string, error) {
+	return "", nil
+}
`

func TestParseDiff(t *testing.T) {
	files, err := ParseDiff(testDiff)
	if err != nil {
		t.Fatalf("ParseDiff returned error: %v", err)
	}

	// Verify 2 files parsed
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// First file checks
	f0 := files[0]
	if f0.NewName != "internal/auth/middleware.go" {
		t.Errorf("first file NewName: expected %q, got %q", "internal/auth/middleware.go", f0.NewName)
	}
	if len(f0.Hunks) != 1 {
		t.Fatalf("first file: expected 1 hunk, got %d", len(f0.Hunks))
	}
	hunk := f0.Hunks[0]
	if hunk.OldStart != 42 {
		t.Errorf("first hunk OldStart: expected 42, got %d", hunk.OldStart)
	}

	var added, removed int
	for _, dl := range hunk.Lines {
		switch dl.Type {
		case LineAdded:
			added++
		case LineRemoved:
			removed++
		}
	}
	if added != 5 {
		t.Errorf("first hunk: expected 5 added lines, got %d", added)
	}
	if removed != 1 {
		t.Errorf("first hunk: expected 1 removed line, got %d", removed)
	}

	// Second file checks
	f1 := files[1]
	if f1.OldName != "/dev/null" {
		t.Errorf("second file OldName: expected %q, got %q", "/dev/null", f1.OldName)
	}
	if f1.NewName != "internal/auth/token.go" {
		t.Errorf("second file NewName: expected %q, got %q", "internal/auth/token.go", f1.NewName)
	}
}
