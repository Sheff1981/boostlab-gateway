package probe

import "testing"

func TestValidPayloadV1(t *testing.T) {
	if !validPayload([]byte(MagicV1)) {
		t.Fatal("expected v1 discovery payload to be valid")
	}
}

func TestValidPayloadV2(t *testing.T) {
	payload := []byte("BOOSTLAB/PROBE/2/0123456789abcdef/7")
	if !validPayload(payload) {
		t.Fatal("expected v2 probe payload to be valid")
	}
}

func TestRejectsMalformedV2(t *testing.T) {
	cases := [][]byte{
		[]byte("BOOSTLAB/PROBE/2/short/1"),
		[]byte("BOOSTLAB/PROBE/2/0123456789abcdef/not-a-number"),
		[]byte("BOOSTLAB/PROBE/2/0123456789abcdef"),
		[]byte("OTHER"),
	}

	for _, payload := range cases {
		if validPayload(payload) {
			t.Fatalf("expected invalid payload: %q", string(payload))
		}
	}
}

func TestProbeResponseCannotAmplify(t *testing.T) {
	payload := []byte("BOOSTLAB/PROBE/2/0123456789abcdef/42")
	if len(payload) > maxProbeBytes {
		t.Fatal("test payload exceeds max probe size")
	}
}
