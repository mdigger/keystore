package keystore

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

// var logger = log.New("DB")

// DB описывает файловое хранилище данных, где значения задаются и выбираются
// с помощью ключа (key-value store).
type DB struct {
	f       *os.File
	indexes map[string]index // map with key and address of values
	deleted []index          // свободные ячейки для записи данных
	counter uint64           // счетчик
	mu      sync.RWMutex     // блокировка одновременного доступа к файлам
	sync    bool             // выполнять принудительный сброс данных в файл при каждой записи
}

// open открывает файл с данными и инициализирует работу с ним.
//
// По умолчанию открытое хранилище использует синхронную запись данных. Если
// необходимо это отменить, то можно воспользоваться методом db.SetSync()
// после открытия хранилища.
func open(filename string) (db *DB, err error) {
	// logger.Debug("open", "filename", filename)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	// закрываем файл в случае ошибки
	defer func() {
		if err != nil {
			_ = file.Close() // закрываем файл
		}
	}()

	// header описывает заголовок файла с индексом и данными
	const signature uint32 = 0xD3EFAA03
	var header = &struct {
		Signature uint32 // заголовок файла
		Counter   uint64 // глобальный счетчик для генерации уникальых значений
	}{
		Signature: signature,
	}
	// если файл только создан, то записываем вначало сигнатуру,
	if info, _ := file.Stat(); info.Size() == 0 {
		// записываем заголовок индекса
		if err = binary.Write(file, binary.BigEndian, header); err != nil {
			return nil, err
		}
		// иначе проверяем, что она там есть и версия совпадает
	} else {
		if err = binary.Read(file, binary.BigEndian, header); err != nil {
			return nil, err
		}
		if header.Signature != signature {
			return nil, &os.PathError{Op: "check", Path: file.Name(),
				Err: errors.New("bad file format")}
		}
	}
	// читаем файл с данными и воспроизводим индекс
	var (
		offset      int64 = 12                     // размер заголовка с счетчиком
		storedIndex       = new(storedIndex)       // сохраненная информация об индексе
		indexes           = make(map[string]index) // список индексов по именами ключей
		deleted           = make([]index, 0, 100)  // список свободных мест
	)
	for {
		// читаем заголовок с индексной информацией
		if err = binary.Read(file, binary.BigEndian, storedIndex); err != nil {
			break
		}
		// читаем имя ключа
		var key = make([]byte, storedIndex.KeySize)
		if _, err = file.Read(key); err != nil {
			break
		}
		var strKey = string(key)
		// инициализируем описание индекса и сохраняем его
		var index = index{
			Offset:    uint32(offset),
			KeySize:   storedIndex.KeySize,
			DataSize:  storedIndex.DataSize,
			EmptySize: storedIndex.EmptySize,
		}
		if !storedIndex.Deleted {
			// на всякий случай, в случае дублирования индекса оставляем
			// только последнюю запись
			if idx, ok := indexes[strKey]; ok {
				// logger.Warn("dublicate", "key", strKey)
				deleted = append(deleted, idx)
			}
			indexes[strKey] = index
		} else {
			deleted = append(deleted, index)
		}
		// logger.Debug("load index", "key", strKey, "index", index, "deleted", storedIndex.Deleted)
		offset += storedIndex.Size() // позиция следующей записи
		// пропускаем данные и возможное свободное пространство за ними
		// и устанавливаем курсор на следующие данные
		if _, err = file.Seek(offset, io.SeekStart); err != nil {
			break
		}
	}
	if err != io.EOF {
		return nil, err
	}
	// сортируем удаленные данные по размеру занимаемого ими места
	sort.Slice(deleted, func(i, j int) bool {
		var s1, s2 = deleted[i].Size(), deleted[j].Size()
		return s1 < s2 || (s1 == s2 && deleted[i].Offset < deleted[j].Offset)
	})
	// возвращаем инициализированное хранилище
	db = &DB{
		f:       file,
		indexes: indexes,
		deleted: deleted,
		counter: header.Counter,
		sync:    true,
	}
	return db, nil
}

// storedIndex описывает формат хранимого индекса.
type storedIndex struct {
	Time      uint32 // timestamp
	Deleted   bool   // флаг удаленного ключа
	KeySize   uint8  // размер ключа
	DataSize  uint32 // размер данных
	EmptySize uint32 // размер свободного места за данными
}

var storedIndexSize = int64(binary.Size(new(storedIndex)))

// Size возвращает размер, реально занимаемый данными, включая индекс.
func (i storedIndex) Size() int64 {
	// плюс размер данных storedIndex
	return storedIndexSize + int64(i.KeySize) + int64(i.DataSize+i.EmptySize)
}

// index описывает данные, хранимые об индексе в памяти.
type index struct {
	Offset    uint32 // смещение от начала файла
	KeySize   uint8  // длина названия ключа
	DataSize  uint32 // размер данных
	EmptySize uint32 // размер свободного места за данными
}

var indexSize = int64(binary.Size(new(index)))

// Size возвращает суммарный размер ключа и данных, но без учета метаданных.
func (i index) Size() uint32 {
	return uint32(i.KeySize) + i.DataSize + i.EmptySize
}

// DataOffset возвращает смещение относительно начала файла для чтения данных.
func (i index) DataOffset() int64 {
	// плюс размер данных storedIndex и размер ключа
	return int64(i.Offset) + indexSize + int64(i.KeySize) + 1
}

// String возвращает строковое представление индекса, используемое для отладки.
func (i index) String() string {
	return fmt.Sprintf("%d:%d", i.Offset, i.Size())
}

// String возвращает имя хранилища.
func (db *DB) String() string {
	return fmt.Sprintf("db:%s", db.f.Name())
}

// Path возвращает имя файла с хранилищем.
func (db *DB) Path() string {
	return db.f.Name()
}

// Sync принудительно сбрасывает данные из кеша в файл.
//
// Вызов данного метода обычно не требуется, если вручную не выключен
// автоматический сброс кешей при любой операции записи.
func (db *DB) Sync() error {
	// logger.Trace("sync")
	return db.f.Sync()
}

// SetSync устанавливает значение флага автоматического сброса кеша после
// каждой записи.
func (db *DB) SetSync(sync bool) {
	db.mu.Lock()
	db.sync = sync
	db.mu.Unlock()
}

// close закрывает файл с данными хранилища.
func (db *DB) close() (err error) {
	if db.f.Fd() == ^(uintptr(0)) {
		return nil // файл уже закрыт
	}
	db.mu.RLock()
	if db.sync {
		err = db.Sync()
	}
	db.mu.RUnlock()
	// logger.Debug("close")
	if err2 := db.f.Close(); err == nil {
		err = err2
	}
	return err
}

// Close закрывает хранилище.
func (db *DB) Close() error {
	mu.Lock()
	delete(dbs, db.f.Name()) // удаляем из списка открытых
	mu.Unlock()
	return db.close()
}

// Count возвращает количество записей в хранилище.
func (db *DB) Count() uint32 {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return uint32(len(db.indexes))
}

// NextSequence возвращает значение счетчика, которое увеличивается при каждом
// обращении к данной функции. Обычно используется для задания гарантированного
// уникального идентификатора записи хранилища, т.к. последнее использованное
// значение сохраняется в хранилище.
func (db *DB) NextSequence() (uint64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.counter++
	var counter = make([]byte, 8)
	binary.BigEndian.PutUint64(counter, db.counter)
	_, err := db.f.WriteAt(counter, 4) // счетчик идет сразу после сигнатуры файла
	if err == nil && db.sync {
		err = db.Sync()
	}
	return db.counter, err
}

// ErrNotFound возвращается, если данные с таким ключом в хранилище не найдены.
var ErrNotFound = errors.New("key not found")

// get возвращает данные, сохраненные с указанным ключом.
func (db *DB) get(key []byte) ([]byte, error) {
	index, ok := db.indexes[string(key)]
	if !ok {
		return nil, ErrNotFound
	}
	var data = make([]byte, index.DataSize)
	_, err := db.f.ReadAt(data, index.DataOffset())
	if err != nil {
		return nil, err
	}
	// logger.Debug("get", "key", string(key), "value", string(data), "index", index)
	return data, nil
}

// Get возвращает данные, сохраненные с указанным ключом. Если данные с таким
// ключем в хранилище не сохранены, то возвращается ошибка ErrNotFound.
func (db *DB) Get(key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.get(key)
}

// GetJSON преобразует значение из хранилища обратно в объект. Возвращает
// ошибку, если данные с таким ключем не сохранены или формат сохраненных
// данных не соответствует формату JSON.
func (db *DB) GetJSON(key []byte, v interface{}) error {
	data, err := db.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Gets возвращает список значений, соответствующих списку ключей. Игнорирует
// ошибки с ненайденными ключами: в этом случае в качестве значения для такого
// ключа будет возвращено nil.
func (db *DB) Gets(keys ...[]byte) (result [][]byte, err error) {
	result = make([][]byte, len(keys))
	db.mu.RLock()
	defer db.mu.RUnlock()
	for i, key := range keys {
		result[i], err = db.get(key)
		if err != nil && err != ErrNotFound {
			return nil, err
		}
	}
	return result, nil
}

// GetsJSON возвращает массив значений для указанных ключей в формате
// json.RawMessage. Возвращает ошибку, если сохраненные данные не соответствуют
// формату JSON. Для тех ключей, для которых не задано значение, возвращается
// nil.
//
// Данную функцию удобно использовать для отдачи результатов выборки в ответ
// на HTTP-запрос.
func (db *DB) GetsJSON(keys ...[]byte) (result []json.RawMessage, err error) {
	result = make([]json.RawMessage, len(keys))
	db.mu.RLock()
	defer db.mu.RUnlock()
	for i, key := range keys {
		data, err := db.get(key)
		if err != nil && err != ErrNotFound {
			return nil, err
		}
		if !json.Valid(data) {
			return nil, fmt.Errorf("invalid JSON format for key %q", key)
		}
		result[i] = json.RawMessage(data)
	}
	return result, nil
}

// Has возвращает true, если значение с таким ключом определено.
func (db *DB) Has(key string) bool {
	db.mu.RLock()
	_, ok := db.indexes[key]
	db.mu.RUnlock()
	return ok
}

// Keys возвращает список ключей, подходящих под запрос.
//
// Для выборки по ключам используется их отсортированный список. Порядок
// сортировки задается параметром asc. Если указан prefix, то будут выбраны
// только те ключи, которые соответствуют этому префиксу. last позволяет выбрать
// только те ключи, которые идут за или перед индексом с этим значением
// (не включая сам элемент last). Ключ со значением last не обязательно должен
// присутствовать в хранилище: в этом случае просто отбрасывается все до того
// места, где бы он мог быть в отсортированном списке. offset задает сдвиг
// относительно начала списка, а limit - ограничивает количество ключей в
// выборке.
func (db *DB) Keys(prefix, last []byte, offset, limit uint32, asc bool) [][]byte {
	db.mu.RLock()
	var keys = make([][]byte, 0, len(db.indexes))
	// выбираем подходящие ключи
	for key := range db.indexes {
		if bytes.HasPrefix([]byte(key), prefix) {
			keys = append(keys, []byte(key))
		}
	}
	db.mu.RUnlock()
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i], keys[j]) < 0 == asc
	})
	if last != nil {
		// находим в списке строку, где оно должна бы была быть
		var found = sort.Search(len(keys), func(i int) bool {
			return bytes.Compare(keys[i], last) >= 0 == asc
		})
		// в случае точного совпадения, исключаем само значение
		if found < len(keys) && bytes.Equal(keys[found], last) {
			found++
		}
		keys = keys[found:] // оставляем только ключи после указанного
	}
	var min = func(x, y int) int {
		if x < y {
			return x
		}
		return y
	}
	if offset > 0 {
		keys = keys[min(int(offset), len(keys)):]
	}
	if limit > 0 {
		keys = keys[:min(int(limit), len(keys))]
	}
	return keys
}

var bufPool = sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}

// delete удаляет ключ из хранилища.
func (db *DB) delete(key []byte) error {
	index, ok := db.indexes[string(key)]
	if !ok {
		return ErrNotFound
	}
	// записиваем в заголовок метку об удалении
	var buf = bufPool.Get().(*bytes.Buffer)
	buf.Reset() // сбрасываем буфер от возможного предыдущего значения
	// записываем только метку об удалении
	_ = binary.Write(buf, binary.BigEndian, &struct {
		Time    uint32 // время удаление
		Deleted bool   // метка об удалении
	}{
		Time:    uint32(time.Now().Unix()),
		Deleted: true,
	})
	_, err := db.f.WriteAt(buf.Bytes(), int64(index.Offset))
	bufPool.Put(buf)
	if err != nil {
		return err
	}
	delete(db.indexes, string(key)) // удаляем информацию об индексе
	// logger.Debug("delete", "key", string(key), "index", index)
	// сохраняем информацию об освободившемся для записи месте
	var dl = len(db.deleted)
	found := sort.Search(dl, func(i int) bool {
		var s1, s2 = db.deleted[i].Size(), index.Size()
		return s1 > s2 || (s1 == s2 && db.deleted[i].Offset > index.Offset)
	})
	if found < dl && db.deleted[found].Offset == index.Offset {
		// logger.Warn("dublicate free", "key", string(key), "index", index)
		return nil // не добавляем дубль
	}
	// https://blog.golang.org/go-slices-usage-and-internals
	db.deleted = append(db.deleted, index) //grow origin slice capacity if needed
	if found < dl {
		copy(db.deleted[found+1:], db.deleted[found:]) //ha-ha, lol, 20x faster
		db.deleted[found] = index
	}
	return nil
}

// Delete удаляет ключ из хранилища. Если значения с таким ключом в хранилище
// нет, то возвращается ошибка ErrNotFound.
func (db *DB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	err := db.delete(key)
	if err == nil && db.sync {
		return db.Sync()
	}
	return err
}

// Deletes удаляет список ключей из хранилища. В отличие от метода Delete, не
// возвращает ошибку об отсуствии ключа в хранилище.
func (db *DB) Deletes(keys ...[]byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, key := range keys {
		if err := db.delete(key); err != nil && err != ErrNotFound {
			return err
		}
	}
	if db.sync {
		return db.Sync()
	}
	return nil
}

// put сохраняет данные в хранилище с указанным ключом.
func (db *DB) put(key, value []byte) (err error) {
	// удаляем индекс, если он существовал, и сводим задачу к первоначальной
	if err := db.delete(key); err != nil && err != ErrNotFound {
		return err
	}
	// теперь находим подходящее место для вставки данных
	var (
		dataSize = uint32(len(key) + len(value)) // размер данных для записи
		dl       = len(db.deleted)               // количество свободных мест
		offset   int64                           // итоговое смещение для записи данных
		empty    uint32                          // размер свободного места за данными
	)
	if found := sort.Search(dl, func(i int) bool {
		return db.deleted[i].Size() >= dataSize
	}); found < dl {
		var index = db.deleted[found] // найдено подходящее свободное место
		offset = int64(index.Offset)  // смещение для записи
		// вычисляем размер свободного места, которое останется после данных
		empty = index.Size() - uint32(len(key)+len(value))
		// удаляем этот индекс из свободного доступа
		db.deleted = append(db.deleted[:found], db.deleted[found+1:]...)
	} else {
		// не найдено подходящего места для записи - записываем в конец файла
		offset, err = db.f.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}
	}
	var index = index{
		Offset:    uint32(offset),
		KeySize:   uint8(len(key)),
		DataSize:  uint32(len(value)),
		EmptySize: empty,
	}
	// записываем заголовок с индексом и сами данные в файл хранилища
	var buf = bufPool.Get().(*bytes.Buffer)
	buf.Reset() // сбрасываем буфер от возможного предыдущего значения
	_ = binary.Write(buf, binary.BigEndian, &storedIndex{
		Time:      uint32(time.Now().Unix()),
		Deleted:   false,
		KeySize:   index.KeySize,
		DataSize:  index.DataSize,
		EmptySize: index.EmptySize,
	})
	_, _ = buf.Write(key)                      // имя ключа
	_, _ = buf.Write(value)                    // данные
	_, err = db.f.WriteAt(buf.Bytes(), offset) // сохраняем в хранилище
	bufPool.Put(buf)
	if err != nil {
		return err
	}
	// сохраняем индекс
	db.indexes[string(key)] = index
	// logger.Debug("put", "key", string(key), "value", string(value), "index", index)
	return nil
}

// Put сохраняет данные в хранилище с указанным ключом. Если данные с таким
// ключом уже были ранее сохранены в хранилище, то они перезаписываются.
func (db *DB) Put(key, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	err := db.put(key, value)
	if err == nil && db.sync {
		return db.Sync()
	}
	return err
}

// PutJSON сохраняет данные в хранилище с указанным ключом в формате JSON.
// Возвращает ошибку, если не удалось преобразовать объект в формат JSON.
func (db *DB) PutJSON(key []byte, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return db.Put(key, data)
}

// Puts позволяет записать сразу несколько значений в хранилище. Данные
// передаются в виде связанного списка: ключ - значение. Т.к. ключем в map
// не может выступать изменяемый массив байт, то значение ключа задается
// в виде строки.
func (db *DB) Puts(values map[string][]byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	for key, value := range values {
		if err := db.put([]byte(key), value); err != nil {
			return err
		}
	}
	if db.sync {
		return db.Sync()
	}
	return nil
}

// PutsJSON сохраняет в хранилище объекты в формате JSON. Возвращает ошибку,
// если не удалось преобразовать объект в формат JSON. При этом те значения,
// которые на момент ошибки уже были сохранены в хранилище, остаются.
func (db *DB) PutsJSON(values map[string]interface{}) error {
	var result = make(map[string][]byte, len(values))
	for key, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		result[key] = data
	}
	return db.Puts(result)
}
