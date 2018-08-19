package keystore

import (
	"encoding/binary"
	"fmt"
)

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
