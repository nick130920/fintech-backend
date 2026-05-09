package entity

import "testing"

func TestTripMemberRoles(t *testing.T) {
	owner := &TripMember{Role: TripMemberRoleOwner}
	admin := &TripMember{Role: TripMemberRoleAdmin}
	member := &TripMember{Role: TripMemberRoleMember}
	viewer := &TripMember{Role: TripMemberRoleViewer}

	if !owner.IsOwner() {
		t.Fatal("owner should report IsOwner true")
	}
	if member.IsOwner() {
		t.Fatal("member should not be owner")
	}

	if !owner.CanManage() || !admin.CanManage() {
		t.Fatal("owner and admin should manage")
	}
	if member.CanManage() || viewer.CanManage() {
		t.Fatal("member and viewer should not manage")
	}

	if !member.CanRegisterExpenses() {
		t.Fatal("member should register expenses")
	}
	if viewer.CanRegisterExpenses() {
		t.Fatal("viewer should not register expenses")
	}
}

func TestTripMemberMatchesUser(t *testing.T) {
	uid := uint(7)
	member := &TripMember{UserID: &uid}
	if !member.MatchesUser(7) {
		t.Fatal("expected to match user 7")
	}
	if member.MatchesUser(8) {
		t.Fatal("should not match user 8")
	}

	ghost := &TripMember{IsGhost: true}
	if ghost.MatchesUser(7) {
		t.Fatal("ghost member should never match a user")
	}
}
