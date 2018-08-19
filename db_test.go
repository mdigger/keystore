package keystore

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mdigger/log"
)

func init() {
	rand.Seed(time.Now().Unix())
	log.SetLevel(log.DEBUG)
}

func randomAsterics() string {
	return strings.Repeat("*", rand.Intn(18))
}

var filename = "test.db"

func TestDB(t *testing.T) {
	os.RemoveAll(filename)
	for p := 1; p < 50; p++ {
		// t.Logf("test %02d", p)
		db, err := Open(filename)
		if err != nil {
			t.Fatal(err)
		}
		var prefix = "abcd"
		var keys = make([]string, 0, 100)
		for i := 1; i < 100; i++ {
			id := fmt.Sprintf("%c%c",
				prefix[rand.Intn(len(prefix))],
				rand.Intn(26)+'a',
			)
			keys = append(keys, id)
		}
		// fmt.Println(keys)

		for i, key := range keys {
			data := fmt.Sprintf("text init %02d:%04d [%s]",
				p, i+1, randomAsterics())
			err = db.PutJSON([]byte(key), data)
			if err == nil {
				var s string
				if err = db.GetJSON([]byte(key), &s); err == nil {
					if data != s {
						t.Fatalf("data init error: %q - %q", data, s)
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
		}

		// log.Info("----------------------------")
		for i := 1; i < 500; i++ {
			key := []byte(keys[rand.Intn(len(keys))])
			if rand.Intn(3) == 0 {
				if err = db.Delete(key); err == ErrNotFound {
					err = nil
				}
			} else {
				data := fmt.Sprintf("text random %02d:%04d [%s]",
					p, i, randomAsterics())
				err = db.PutJSON(key, data)
				if err == nil {
					var s string
					if err = db.GetJSON(key, &s); err == nil {
						if data != s {
							t.Fatalf("data replace error: %q - %q", data, s)
						}
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
		}

		// log.Info("===============================")
		nkeys := db.Keys(nil, nil, 0, 0, true)
		j, err := db.GetsJSON(nkeys...)
		if err != nil {
			t.Fatal(err)
		}
		_ = j
		// enc := json.NewEncoder(os.Stdout)
		// enc.SetIndent("", "  ")
		// if err := enc.Encode(j); err != nil {
		// 	t.Fatal(err)
		// }
		// fmt.Printf("%s\n", j)
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		// log.Info("++++++++++++++++++++++++")
	}

	db, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	nkeys := db.Keys(nil, nil, 0, 0, true)
	fmt.Printf("keys: %s\n", nkeys)
	j, err := db.GetsJSON(nkeys...)
	if err != nil {
		t.Fatal(err)
	}
	_ = j
	CloseAll()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}
