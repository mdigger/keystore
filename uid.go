package keystore

import (
	"encoding/binary"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	// minDate задает минимальную дату, от которой идет отсчет счетчика
	minDate = time.Unix(1136239445, 0) // 2006-01-02T15:04:05Z07:00
	// globalCounter содержит текущее значение счетчика, которое увеличивается
	// после каждого использования
	globalCounter uint32
)

func init() {
	rand.Seed(time.Now().UnixNano())
	globalCounter = rand.Uint32() // устанавливаем случайное начальное значение
}

// UID представляет из себя уникальный идентификатор, основанный на временной
// метке и внутреннем счетчике. Его удобно использовать в качестве глобального
// уникального идентификатора сразу для нескольких хранилищ, так как его
// значения действительно будут уникальными и монотонно возрастающими, что
// позволяет использовать сортировку ключей в запросах. В качестве точки
// отсчета используется дата 2006-01-02T15:04:05Z07:00.
type UID uint64

// NewUID возвращает уникальный идентификатор, основанный на времени и
// внутреннем счетчике. Шесть байт отведено под текущее время, и два байта -
// под счетчик.
func NewUID() UID {
	var counter = uint16(atomic.AddUint32(&globalCounter, 1))
	return UID(uint64(time.Since(minDate)&^0xffff) + uint64(counter))
}

// DateUID возвращает уже не совсем уникальный идентификатор для указанных
// даты и времени, но без учета счетчика. Может использоваться для выборки
// ключей до или после указанной даты.
//
// Даты до 2006-01-02T15:04:05Z07:00 считаются невалидными и используются
// как нулевые значения, чтобы не нарушать порядок сортировки.
func DateUID(date time.Time) UID {
	if date.IsZero() || date.Before(minDate) {
		date = minDate
	}
	return UID(uint64(date.Sub(minDate) &^ 0xffff))
}

// Byte возвращает бинарное представление уникального идентификатора.
func (uid UID) Byte() []byte {
	bs, _ := uid.MarshalText()
	return bs
}

// String возвращает строковое представление уникального идентификатора.
func (uid UID) String() string {
	return string(uid.Byte())
}

// ParseUID разбирает уникальный идентификатор из строки. В случае ошибки
// разбора будет возвращен 0.
func ParseUID(s string) UID {
	var uid UID
	_ = (&uid).UnmarshalText([]byte(s))
	return uid
}

// Time возвращает информацию о времени создания уникального идентификатора.
func (uid UID) Time() time.Time {
	return minDate.Add(time.Duration(uid &^ 0xffff))
}

// Counter возвращает значение счетчика уникального идентификатора.
func (uid UID) Counter() uint16 {
	return uint16(uid)
}

// MarshalText обеспечивает представление уникального идентификатора в виде
// текста.
func (uid UID) MarshalText() (text []byte, err error) {
	return []byte(strconv.FormatUint(uint64(uid), 36)), nil
}

// UnmarshalText восстанавливает значение уникального идентификатора из
// текстового представления.
func (uid *UID) UnmarshalText(data []byte) error {
	id, err := strconv.ParseUint(string(data), 36, 64)
	if err != nil {
		return err
	}
	*uid = UID(id)
	return nil
}

// MarshalBinary возвращает бинарное представление уникального идентификатора.
func (uid UID) MarshalBinary() ([]byte, error) {
	var data = make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(uid))
	return data, nil
}

// UnmarshalBinary восстанавливает уникальный идентификатор из его бинарного
// представления.
func (uid *UID) UnmarshalBinary(data []byte) error {
	*uid = UID(binary.BigEndian.Uint64(data))
	return nil
}
