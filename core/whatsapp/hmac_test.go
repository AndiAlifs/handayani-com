package whatsapp

import "testing"

func TestVerifyHMACDocsVector(t *testing.T) {
	body := []byte(`{"event":"message","session":"default","engine":"WEBJS"}`)
	const sig = "208f8a55dde9e05519e898b10b89bf0d0b3b0fdf11fdbf09b6b90476301b98d8097c462b2b17a6ce93b6b47a136cf2e78a33a63f6752c2c1631777076153fa89"
	if !VerifyHMAC(body, sig, "my-secret-key") {
		t.Fatal("valid signature rejected")
	}
	if VerifyHMAC(body, "deadbeef", "my-secret-key") {
		t.Fatal("invalid signature accepted")
	}
	if VerifyHMAC(body, sig, "wrong-key") {
		t.Fatal("wrong key accepted")
	}
}

func TestAckToStatus(t *testing.T) {
	cases := map[int]string{-1: "failed", 0: "pending", 1: "sent", 2: "delivered", 3: "read", 4: "read", 99: ""}
	for ack, want := range cases {
		if got := AckToStatus(ack); got != want {
			t.Errorf("AckToStatus(%d) = %q, want %q", ack, got, want)
		}
	}
}
