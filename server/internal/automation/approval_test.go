package automation

import "testing"

func TestMayApproveSeparationOfDuties(t *testing.T) {
	cases := []struct {
		role, approver, trigger string
		want                    bool
	}{
		{"OPERATOR", "a@x", "a@x", false}, // operator cannot approve own run
		{"OPERATOR", "b@x", "a@x", true},
		{"ADMIN", "a@x", "a@x", true}, // single-admin installs must still work
	}
	for _, c := range cases {
		if got := mayApprove(c.role, c.approver, c.trigger); got != c.want {
			t.Errorf("mayApprove(%s,%s,%s)=%v want %v", c.role, c.approver, c.trigger, got, c.want)
		}
	}
}
