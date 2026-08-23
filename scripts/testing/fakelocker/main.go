// Command fakelocker serves the in-memory fake S3 used by the CLI journeys
// on a real TCP address, so two physical machines can share one "locker"
// against a locally run control plane whose storage provider is a fake.
//
// It is a lab fixture, not a product: every bucket name is served (each from its own in-memory store), every
// access key id with the given prefix is accepted, and nothing is persisted.
//
//	go run ./scripts/testing/fakelocker -addr 0.0.0.0:9000 -accept FAKEKEY
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9000", "listen address")
	accept := flag.String("accept", "FAKEKEY", "access key id prefix to accept (hopd's fake provider mints FAKEKEYnnnn)")
	flag.Parse()
	fake := &s3test.Fake{Store: memory.New(), Valid: map[string]bool{}, RejectAs: "ExpiredToken", AcceptPrefix: *accept, AnyBucket: true}
	logged := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		fake.ServeHTTP(w, r)
	})
	srv := &http.Server{Addr: *addr, Handler: logged, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("fake locker listening on %s (accepting access key ids with prefix %q, any bucket, nothing persisted)", *addr, *accept)
	log.Fatal(srv.ListenAndServe())
}
