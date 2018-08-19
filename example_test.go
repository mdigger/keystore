package keystore_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mdigger/keystore"
)

func init() {
	os.RemoveAll("db/") // удаляем каталог с файлами хранилища
}

func Example() {
	// автоматически закрыть по окончании все открытые хранилища
	defer keystore.CloseAll()
	var dbname = "db/test.db" // имя файла с хранилищем данных
	// сохраняем данные в хранилище в формате JSON
	// если такого файла с хранилищем не существует, то он будет создан
	// автоматически
	err := keystore.PutsJSON(dbname, map[string]interface{}{
		"t1": "text message",
		"t2": 24,
		"t3": time.Date(1971, time.December, 24, 23, 0, 0, 0, time.UTC),
		"t4": &struct {
			Text string `json:"text"`
		}{
			Text: "test",
		},
	})
	if err != nil {
		log.Fatal("PutsJSON error:", err)
	}
	// выбираем все ключи, сохраненные в хранилище, которые начинаются на `t`
	keys, err := keystore.Keys(dbname, []byte{'t'}, nil, 0, 0, true)
	if err != nil {
		log.Fatal("Keys error:", err)
	}
	// получаем список значений из выборки по ключам
	result, err := keystore.GetsJSON(dbname, keys...)
	if err != nil {
		log.Fatal("GetsJSON error:", err)
	}
	// выводим выбранные значения в консоль в виде JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	err = enc.Encode(result)
	if err != nil {
		log.Fatal("JSON encode error:", err)
	}
	// output:
	// [
	//   "text message",
	//   24,
	//   "1971-12-24T23:00:00Z",
	//   {
	//     "text": "test"
	//   }
	// ]
}

func ExampleDB_Keys() {
	// открываем хранилище
	db, err := keystore.Open("db/test_keys.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close() // закрываем по окончании
	// заносим тестовые данные
	err = db.PutsJSON(map[string]interface{}{
		"test1": 1,
		"test2": 2,
		"test3": 3,
		"test4": 4,
		"test5": 5,
		"aaaa6": 6,
	})
	if err != nil {
		log.Fatal(err)
	}

	// выбираем все ключи, которые начинаются на `test`
	keys := db.Keys([]byte("test"), nil, 0, 0, true)
	fmt.Printf("1: %q\n", keys)
	// выбираем все ключи, которые начинаются на `test`, но после ключа `test2`
	keys = db.Keys([]byte("test"), []byte("test2"), 0, 0, true)
	fmt.Printf("2: %q\n", keys)
	// сортируем вывод в обратном порядке
	keys = db.Keys(nil, nil, 0, 0, false)
	fmt.Printf("3: %q\n", keys)
	// выбираем не более двух ключей
	keys = db.Keys([]byte("test"), []byte("test2"), 0, 2, true)
	fmt.Printf("4: %q\n", keys)
	// не используем префикс ключа, а выбираем по всем
	keys = db.Keys(nil, []byte("test3"), 0, 0, false)
	fmt.Printf("5: %q\n", keys)
	// output:
	// 1: ["test1" "test2" "test3" "test4" "test5"]
	// 2: ["test3" "test4" "test5"]
	// 3: ["test5" "test4" "test3" "test2" "test1" "aaaa6"]
	// 4: ["test3" "test4"]
	// 5: ["test2" "test1" "aaaa6"]
}
