package fsx

import "testing"

// TestOwnerOnlyDACLAcceptsOnlyProtectedPrivateDescriptors covers the Windows
// SDDL contract from every build platform: OwnerOnly must reject an inheriting
// DACL, a deny entry, and any trustee outside the private set.
func TestOwnerOnlyDACLAcceptsOnlyProtectedPrivateDescriptors(t *testing.T) {
	t.Parallel()

	const user = "S-1-5-21-1111111111-2222222222-3333333333-1001"
	allowed := map[string]bool{"SY": true, "BA": true, user: true}
	tests := []struct {
		name string
		sddl string
		want bool
	}{
		{
			name: "protected private dacl",
			sddl: "O:" + user + "G:BA" + "D:P(A;;FA;;;" + user + ")(A;;FA;;;SY)(A;;FA;;;BA)",
			want: true,
		},
		{
			name: "protected and auto inherited flags",
			sddl: "D:PAI(A;;FA;;;" + user + ")",
			want: true,
		},
		{
			name: "inheritance not blocked",
			sddl: "D:AI(A;;FA;;;" + user + ")",
		},
		{
			name: "everyone granted access",
			sddl: "D:P(A;;FA;;;" + user + ")(A;;FA;;;WD)",
		},
		{
			name: "other local account granted access",
			sddl: "D:P(A;;FA;;;S-1-5-21-1111111111-2222222222-3333333333-1002)",
		},
		{
			name: "non allow entry",
			sddl: "D:P(D;;FA;;;" + user + ")(A;;FA;;;SY)",
		},
		{
			name: "empty protected dacl",
			sddl: "D:P",
		},
		{
			name: "no dacl section",
			sddl: "O:" + user + "G:BA",
		},
		{
			name: "malformed entry",
			sddl: "D:P(A;;FA;;;",
		},
		{
			name: "truncated entry fields",
			sddl: "D:P(A;;FA)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, detail := ownerOnlyDACL(test.sddl, allowed)
			if got != test.want {
				t.Fatalf("ownerOnlyDACL(%q) = %t (%s), want %t", test.sddl, got, detail, test.want)
			}
			if detail == "" {
				t.Fatalf("ownerOnlyDACL(%q) returned no detail", test.sddl)
			}
		})
	}
}
