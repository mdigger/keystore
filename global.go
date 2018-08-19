package keystore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var (
	dbs = make(map[string]*DB) // коллекция открытых хранилищ
	mu  sync.Mutex             // блокировщик доступа
)

// Open возвращает открытую базу с хранилищем в указанном файле. Если база уже
// была открыта, то повторного открытия не происходит, а возвращается ссылка на
// ранее открытую.
//
// При первом открытии файла происходит построение индекса ключей и проверка
// целостности данных, в процессе чего файл читается от начала и до конца.
// При очень больших файлах данных это может занимать некоторое время, поэтому
// не рекомендуется использовать эту библиотеку для хранения большого
// количества данных.
//
// По умолчанию хранилище открывается в синхронном режиме: т.е. любая запись
// в хранилище приводит к принудительному сбросу данных в файл, что сильно
// замедляет работу. Если вы хотите самостоятельно управлять процессом сброса
// кеша или довериться операционной системе, то используйте вызов метода
// db.SetSync(false).
func Open(filename string) (db *DB, err error) {
	mu.Lock()
	defer mu.Unlock()
	db, ok := dbs[filename]
	if !ok {
		// создаем каталог, если он еще не создан
		if dir := filepath.Dir(filename); dir != "." {
			err = os.MkdirAll(dir, 0777)
			if err != nil {
				return nil, err
			}
		}
		db, err = open(filename)
		if err != nil {
			return nil, err
		}
		dbs[filename] = db
	}
	return db, nil
}

// Close закрывает хранилище с указанным именем. Не возвращает ошибку, если
// хранилище не было открыто.
func Close(filename string) error {
	mu.Lock()
	defer mu.Unlock()
	if db, ok := dbs[filename]; ok {
		delete(dbs, filename)
		return db.close()
	}
	return nil
}

// CloseAll закрывает все открытые хранилища. Ошибка закрытия хранилищ не
// обрабатывается.
func CloseAll() {
	mu.Lock()
	for name, db := range dbs {
		delete(dbs, name)
		db.close()
	}
	mu.Unlock()
}

// DeleteDB удаляет файл с хранилищем с заданным именем, предварительно его
// закрывая, если оно было открыто.
func DeleteDB(filename string) error {
	if err := Close(filename); err != nil {
		return err
	}
	return os.Remove(filename)
}

// Count возвращает количество записей в хранилище.
func Count(filename string) (uint32, error) {
	db, err := Open(filename)
	if err != nil {
		return 0, err
	}
	return db.Count(), nil
}

// NextSequence возвращает значение счетчика, которое увеличивается при каждом
// обращении к данной функции хранилища.
func NextSequence(filename string) (uint64, error) {
	db, err := Open(filename)
	if err != nil {
		return 0, err
	}
	return db.NextSequence()
}

// Get возвращает данные, сохраненные с указанным ключом. Если данных с таким
// ключем в хранилище нет, то возвращается ошибка ErrNotFound.
func Get(filename string, key []byte) ([]byte, error) {
	db, err := Open(filename)
	if err != nil {
		return nil, err
	}
	return db.Get(key)
}

// GetJSON преобразует значение из хранилища объект. Значение в хранилище
// должно быть представлено в формате JSON, иначе вернется ошибка.
func GetJSON(filename string, key []byte, v interface{}) error {
	db, err := Open(filename)
	if err != nil {
		return err
	}
	return db.GetJSON(key, v)
}

// Gets возвращает список значений, соответствующих списку ключей. Игнорирует
// ошибки с ненайденными ключами: в этом случае в качестве значения для данного
// ключа будет возвращен nil.
func Gets(filename string, keys ...[]byte) (result [][]byte, err error) {
	db, err := Open(filename)
	if err != nil {
		return nil, err
	}
	return db.Gets(keys...)
}

// GetsJSON возвращает массив значений для указанных ключей в формате
// json.RawMessage. Возвращает ошибку, если данные не соответствуют формату
// JSON. Для ненайденных ключей возвращается значение nil.
func GetsJSON(filename string, keys ...[]byte) (result []json.RawMessage, err error) {
	db, err := Open(filename)
	if err != nil {
		return nil, err
	}
	return db.GetsJSON(keys...)
}

// Has возвращает true, если значение с таким ключом задано в хранилище.
func Has(filename string, key string) (bool, error) {
	db, err := Open(filename)
	if err != nil {
		return false, err
	}
	return db.Has(key), nil
}

// Keys возвращает список ключей, подходящих под запрос.
//
// Подробную информацию по параметрам смотри в описании метода db.Keys.
func Keys(filename string, prefix, last []byte, offset, limit uint32, asc bool) ([][]byte, error) {
	db, err := Open(filename)
	if err != nil {
		return nil, err
	}
	return db.Keys(prefix, last, offset, limit, asc), nil
}

// Delete удаляет значение с указанным ключом из хранилища.
func Delete(filename string, key []byte) error {
	db, err := Open(filename)
	if err != nil {
		return err
	}
	return db.Delete(key)
}

// Deletes удаляет список ключей из хранилища. В отличие от метода Delete,
// позволяет удалить более одного ключа сразу и не возвращает ошибку при
// отсуствии ключа в хранилище.
func Deletes(filename string, keys ...[]byte) error {
	db, err := Open(filename)
	if err != nil {
		return err
	}
	return db.Deletes(keys...)
}

// Put сохраняет данные в хранилище с указанным ключом. Если данные с таким
// ключом уже были сохранены в хранилище, то они удаляются и перезаписываются
// на новые.
func Put(filename string, key, value []byte) error {
	db, err := Open(filename)
	if err != nil {
		return err
	}
	return db.Put(key, value)
}

// PutJSON сохраняет данные в хранилище с указанным ключом в формате JSON.
// Возвращает ошибку, если не удалось преобразовать объект в формат JSON.
func PutJSON(filename string, key []byte, value interface{}) error {
	db, err := Open(filename)
	if err != nil {
		return err
	}
	return db.PutJSON(key, value)
}

// Puts позволяет записать сразу несколько значений в хранилище. Для передачи
// списка данных используется словарь с именем ключа и связанным с ним
// значением.
func Puts(filename string, values map[string][]byte) error {
	db, err := Open(filename)
	if err != nil {
		return err
	}
	return db.Puts(values)
}

// PutsJSON сохраняет в хранилище объекты в формате JSON. В случае невозможности
// представления объекта в виде JSON, возвращается ошибка. При этом те записи,
// которые на тот момент уже успели записаться, сохраняются.
func PutsJSON(filename string, values map[string]interface{}) error {
	db, err := Open(filename)
	if err != nil {
		return err
	}
	return db.PutsJSON(values)
}
