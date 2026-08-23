package s3

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

// TestListFollowsContinuationTokens covers ticket #13's migration listing:
// S3 caps ListObjectsV2 at 1000 keys per page, and a locker with a long
// history has more, so every page must be followed.
func TestListFollowsContinuationTokens(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		objects  int
		prefix   string
	}{
		{name: "single page", pageSize: 0, objects: 5, prefix: ""},
		{name: "exact multiple of page", pageSize: 2, objects: 6, prefix: ""},
		{name: "partial last page", pageSize: 2, objects: 7, prefix: ""},
		{name: "client prefix and list prefix", pageSize: 3, objects: 7, prefix: "profiles/p1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeS3(t, "reinstate")
			f.accept("AKIA1")
			f.Mu.Lock()
			f.PageSize = tc.pageSize
			f.Mu.Unlock()
			c := f.client(t, Config{Credentials: Static("AKIA1", "s"), Prefix: tc.prefix})
			ctx := context.Background()
			var want []string
			for i := 0; i < tc.objects; i++ {
				key := fmt.Sprintf("snapshots/%03d.age", i)
				if _, err := c.Put(ctx, key, strings.NewReader("x"), 1, backend.PutOptions{}); err != nil {
					t.Fatal(err)
				}
				want = append(want, key)
			}
			if _, err := c.Put(ctx, "manifest.json", strings.NewReader("{}"), 2, backend.PutOptions{}); err != nil {
				t.Fatal(err)
			}
			got, err := c.List(ctx, "snapshots/")
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			for _, o := range got {
				keys = append(keys, o.Key)
			}
			sort.Strings(keys)
			if strings.Join(keys, ",") != strings.Join(want, ",") {
				t.Fatalf("listed %v, want %v", keys, want)
			}
			lists := 0
			for _, r := range f.requestLog() {
				if strings.HasPrefix(r, "GET  as") || strings.HasPrefix(r, "GET as") {
					lists++
				}
			}
			wantLists := 1
			if tc.pageSize > 0 {
				wantLists = (tc.objects + tc.pageSize - 1) / tc.pageSize
			}
			if lists != wantLists {
				t.Fatalf("issued %d list requests, want %d:\n%s", lists, wantLists, strings.Join(f.requestLog(), "\n"))
			}
		})
	}
}
