package credentials

import (
	"errors"
	"testing"

	keyring "github.com/zalando/go-keyring"
)

func TestDeviceTokenKeyringRoundTrip(t *testing.T) {
	keyring.MockInit()
	store := NewKeyringStore()
	if _, err := store.GetDeviceToken(); !errors.Is(err, ErrNoDeviceToken) {
		t.Fatalf("err=%v want ErrNoDeviceToken", err)
	}
	want := DeviceToken{Token: "hop_abc", ControlPlaneURL: "https://hop.reinstate.dev", AccountID: "a", DeviceID: "d"}
	if err := store.SetDeviceToken(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetDeviceToken()
	if err != nil || got != want {
		t.Fatalf("got %+v err=%v", got, err)
	}
	if err := store.DeleteDeviceToken(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetDeviceToken(); !errors.Is(err, ErrNoDeviceToken) {
		t.Fatalf("after delete err=%v", err)
	}
	if err := store.DeleteDeviceToken(); err != nil {
		t.Fatalf("second delete: %v", err)
	}
	if err := store.SetDeviceToken(DeviceToken{Token: "x"}); err == nil {
		t.Fatal("incomplete token accepted")
	}
}
