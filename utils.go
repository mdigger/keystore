package keystore

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// Bytes преобразует данные в бинарный формат с помощью binary.BigEndian.
// Отдельная обработка добавлена для string, []byte,  json.RawMessage, byte
// и всех остальных, кто поддерживает encoding.BinaryMarshaler,
// encoding.TextMarshaler, json.Marshaler или fmt.Stringer. Возвращает ошибку,
// если преобразование не получилось.
func Bytes(v interface{}) ([]byte, error) {
	switch v := v.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	case json.RawMessage:
		return []byte(v), nil
	case byte:
		return []byte{v}, nil
	case encoding.BinaryMarshaler:
		return v.MarshalBinary()
	case encoding.TextMarshaler:
		return v.MarshalText()
	case json.Marshaler:
		return v.MarshalJSON()
	case fmt.Stringer:
		return []byte(v.String()), nil
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
