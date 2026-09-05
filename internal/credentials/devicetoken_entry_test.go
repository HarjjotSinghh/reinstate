package credentials

import (
	"runtime"
	"strings"
	"testing"
)

func TestDeviceTokenEntryKeepsTheDefaultHomeOnTheLegacyEntry(t *testing.T) {
	for _, home := range []string{"", "   ", "\t"} {
		if got := deviceTokenEntry(home); got != DeviceTokenRef {
			t.Fatalf("deviceTokenEntry(%q) = %q, want %q", home, got, DeviceTokenRef)
		}
	}
}

func TestDeviceTokenEntryGivesEveryOtherHomeItsOwnEntry(t *testing.T) {
	a := deviceTokenEntry(`D:\lab\device-a\home`)
	b := deviceTokenEntry(`D:\lab\device-b\home`)
	if a == DeviceTokenRef || b == DeviceTokenRef {
		t.Fatalf("an overridden home must not share the default entry: a=%q b=%q", a, b)
	}
	if a == b {
		t.Fatalf("two homes collapsed onto one entry: %q", a)
	}
	if !strings.HasPrefix(a, DeviceTokenRef+"@") {
		t.Fatalf("entry %q does not extend the legacy name", a)
	}
	if again := deviceTokenEntry(`D:\lab\device-a\home`); again != a {
		t.Fatalf("entry is not stable: %q then %q", a, again)
	}
}

func TestDeviceTokenEntryIgnoresTrailingSeparators(t *testing.T) {
	if deviceTokenEntry(`D:\lab\home`) != deviceTokenEntry(`D:\lab\home\`) {
		t.Fatal("a trailing separator changed the entry")
	}
	if deviceTokenEntry("/home/dev/.reinstate") != deviceTokenEntry("/home/dev/.reinstate/") {
		t.Fatal("a trailing slash changed the entry")
	}
}

func TestDeviceTokenEntryIsCaseInsensitiveOnWindowsOnly(t *testing.T) {
	same := deviceTokenEntry(`D:\Lab\Home`) == deviceTokenEntry(`d:\lab\home`)
	if runtime.GOOS == "windows" && !same {
		t.Fatal("Windows paths differing only in case must map to one entry")
	}
	if runtime.GOOS != "windows" && same {
		t.Fatal("case must distinguish homes on a case-sensitive filesystem")
	}
}

func TestDeviceTokenEntryReadsTheHomeOverride(t *testing.T) {
	t.Setenv(deviceTokenHomeEnv, "")
	if got := DeviceTokenEntry(); got != DeviceTokenRef {
		t.Fatalf("with no override: %q, want %q", got, DeviceTokenRef)
	}
	t.Setenv(deviceTokenHomeEnv, t.TempDir())
	if got := DeviceTokenEntry(); got == DeviceTokenRef {
		t.Fatal("with an override the default entry was still used")
	}
}
