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
	minDate = time.Date(2018, time.August, 0, 0, 0, 0, 0, time.UTC)
	// globalCounter содержит текущее значение счетчика, которое увеличивается
	// после каждого использования
	globalCounter uint32
)

func init() {
	rand.Seed(time.Now().UnixNano())
	globalCounter = rand.Uint32() // устанавливаем случайное начальное значение
}

// UID представляет из себя уникальный идентификатор, основанный на временной
// метке и внутреннем счетчике.
type UID uint64

// NewUID возвращает уникальный идентификатор, основанный на времени и
// внутреннем счетчике. Шесть байт отведено под текущее время, и два байта -
// под счетчик.
func NewUID() UID {
	var counter = uint16(atomic.AddUint32(&globalCounter, 1))
	return UID(uint64(time.Since(minDate)&^0xffff) + uint64(counter))
}

const uidbase = 36

// String возвращает строковое представление уникального идентификатора.
func (uid UID) String() string {
	return strconv.FormatUint(uint64(uid), uidbase)
}

// ParseUID разбирает уникальный идентификатор из строки.
func ParseUID(s string) (UID, error) {
	id, err := strconv.ParseUint(s, uidbase, 64)
	if err != nil {
		return 0, err
	}
	return UID(id), nil
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
	return []byte(uid.String()), nil
}

// UnmarshalText восстанавливает значение уникального идентификатора из
// текстового представления.
func (uid *UID) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*uid = 0
		return nil
	}
	id, err := strconv.ParseUint(string(text), uidbase, 64)
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
