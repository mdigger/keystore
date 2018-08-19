package keystore

import (
	"fmt"
	"testing"
	"time"
)

func TestUID(t *testing.T) {
	for i := 0; i < 25; i++ {
		uid := NewUID()
		fmt.Printf("%2d - %s [%[2]d] %d, %v\n", i, uid, uid.Counter(), uid.Time())
		uid2 := ParseUID(uid.String())
		if uid2 != uid {
			t.Errorf("%2d - bad ParseUID: %d vs %d", i, uid, uid2)
		}
		text, err := uid.MarshalText()
		if err != nil {
			t.Error(err)
		}
		var uid3 UID
		err = uid3.UnmarshalText(text)
		if err != nil {
			t.Error(err)
		}
		if uid3 != uid {
			t.Errorf("%2d - bad MarshalText uid: %d vs %d", i, uid, uid3)
		}

		text, err = uid.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		var uid4 UID
		err = uid4.UnmarshalBinary(text)
		if err != nil {
			t.Error(err)
		}
		if uid4 != uid {
			t.Errorf("%2d - bad MarshalBinary uid: %d vs %d", i, uid, uid4)
		}
	}
}

func TestDateUID(t *testing.T) {
	if DateUID(minDate) != 0 {
		t.Fatal("bad UID for min date")
	}
	// fmt.Println(DateUID(minDate.Add(-time.Hour)))
	for i := 0; i < 20; i++ {
		uid := DateUID(minDate.Add(time.Hour * 12 * time.Duration(i)))
		fmt.Println(uid, uid.Counter(), uid.Time())
	}
}
