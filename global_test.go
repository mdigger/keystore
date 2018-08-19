package keystore

import (
	"strconv"
	"testing"
)

func TestGlobal(t *testing.T) {
	var dbs = []string{"db/t1.db", "db/t2.db"}
	if err := OpenAll(dbs...); err != nil {
		t.Fatal(err)
	}
	defer CloseAll()

	id, err := NextSequence(dbs[1])
	if err != nil {
		t.Error(err)
	}
	err = Put(dbs[1], strconv.FormatUint(id, 10), `"p1"`)
	if err != nil {
		t.Error(err)
	}
	err = Puts(dbs[1], map[string]interface{}{
		"k1": `"k1"`,
		"k2": `"k2"`,
		"k3": `"k3"`,
	})
	if err != nil {
		t.Error(err)
	}
	err = PutsJSON(dbs[1], map[string]interface{}{
		"j1": "j1",
		"j2": "j2",
		"j3": "j3",
	})
	err = PutJSON(dbs[1], "j4", "j4")
	if err != nil {
		t.Error(err)
	}

	count, err := Count(dbs[1])
	if err != nil {
		t.Error(err)
	}
	if count != 8 {
		t.Error("bad record count")
	}
	// fmt.Println("count:", count)

	h, err := Has(dbs[1], "j4")
	if err != nil {
		t.Error(err)
	}
	if !h {
		t.Error("bad Has method")
	}

}
