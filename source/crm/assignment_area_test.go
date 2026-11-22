package crm

import "testing"

func TestAssignWithExplicitCity(t *testing.T) {
	j := fixture(t)
	j.Status = "new"
	j.Area = ""
	j.Address = "Maitikkakuja 6"
	n, e := Apply(j, Action{Kind: "assign", Revision: j.Revision, Mechanic: "m", Area: "хельсинки"}, testOwner, testNow)
	if e != nil || n.Area != "Helsinki" || n.Mechanic != "m" {
		t.Fatal(n, e)
	}
	if e = CheckAvailability(testMechanic, n, false); e != nil {
		t.Fatal(e)
	}
	n.Area = "Espoo"
	if e = CheckAvailability(testMechanic, n, false); e == nil {
		t.Fatal("outside mechanic area accepted")
	}
}
func TestContactAddressEditAndConfirmedGuard(t *testing.T) {
	j := fixture(t)
	j.Status = "new"
	address := "New street 1, Helsinki"
	n, e := Apply(j, Action{Kind: "contact", Revision: j.Revision, Address: &address, Area: "Helsinki"}, testOwner, testNow)
	if e != nil || n.Address != address {
		t.Fatal(n, e)
	}
	j.Status = "scheduled"
	if _, e = Apply(j, Action{Kind: "contact", Revision: j.Revision, Address: &address, Area: j.Area}, testOwner, testNow); e == nil {
		t.Fatal("confirmed address changed without rescheduling")
	}
}
