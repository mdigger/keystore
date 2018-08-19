package keystore

import (
	"fmt"
	"testing"
)

func TestUID(t *testing.T) {
	for i := 0; i < 20; i++ {
		uid := NewUID()
		fmt.Println(uid, uid.Counter(), uid.Time())
		uid2, err := ParseUID(uid.String())
		if err != nil {
			t.Error(err)
		}
		if uid2 != uid {
			t.Errorf("bad ParseUID: %q vs %q", uid, uid2)
		}

		text, err := uid.MarshalText()
		if err != nil {
			t.Error(err)
		}
		err = uid2.UnmarshalText(text)
		if err != nil {
			t.Error(err)
		}
		if uid2 != uid {
			t.Errorf("bad MarshalText uid: %q vs %q", uid, uid2)
		}

		text, err = uid.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		err = uid2.UnmarshalBinary(text)
		if err != nil {
			t.Error(err)
		}
		if uid2 != uid {
			t.Errorf("bad MarshalBinary uid: %q vs %q", uid, uid2)
		}
	}
}
