package keystore

import "testing"

func TestTruncate(t *testing.T) {
	var filename = "db/truncate.db"
	db, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer Remove(filename)

	var id = "id0"
	err = db.Put(id, []byte(`test`))
	if err != nil {
		t.Fatal(err)
	}
	err = db.Delete(id)
	if err != nil {
		t.Fatal(err)
	}
}
