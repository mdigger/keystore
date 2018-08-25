package keystore

import (
	"encoding/json"
	"time"
)

// Timestamp подменяет представление времени в формате JSON в виде числа.
type Timestamp struct {
	time.Time
}

// MarshalJSON представляет время в формате JSON в виде числа.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Unix())
}

// UnmarshalJSON десериализует представление времени из формата JSON.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var i int64
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}
	t.Time = time.Unix(i, 0)
	return nil
}
