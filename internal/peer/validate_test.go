package peer

import "testing"

const validPublicKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestValidateIPv4Peer(t *testing.T) {
	err := Validate(Spec{
		Interface: "wg0",
		PublicKey: validPublicKey,
		Address:   "10.77.0.2/32",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRejectsBroadIPv4PeerNetwork(t *testing.T) {
	err := Validate(Spec{
		Interface: "wg0",
		PublicKey: validPublicKey,
		Address:   "10.77.0.0/24",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
