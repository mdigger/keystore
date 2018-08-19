package keystore

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
)

// Bytes преобразует данные в бинарный формат с помощью binary.BigEndian.
// Отдельная обработка добавлена для string, []byte,  json.RawMessage, uint8 и
// int8. Возвращает ошибку, если преобразование не получилось.
func Bytes(v interface{}) ([]byte, error) {
	switch v := v.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case json.RawMessage:
		return []byte(v), nil
	case uint8:
		return []byte{v}, nil
	case int8:
		return []byte{uint8(v)}, nil
	default:
		var buf = bufPool.Get().(*bytes.Buffer)
		buf.Reset() // сбрасываем буфер от возможного предыдущего значения
		defer bufPool.Put(buf)
		err := binary.Write(buf, binary.BigEndian, v)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}
