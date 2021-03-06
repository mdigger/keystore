# Простейшее key-value хранилище для Golang

[![GoDoc](https://godoc.org/github.com/mdigger/keystore?status.svg)](https://godoc.org/github.com/mdigger/keystore)
[![Build Status](https://travis-ci.org/mdigger/keystore.svg?branch=master)](https://travis-ci.org/mdigger/keystore)
[![Coverage Status](https://coveralls.io/repos/github/mdigger/keystore/badge.svg?branch=master)](https://coveralls.io/github/mdigger/keystore?branch=master)

Все данные хранятся в одном файле и не используют отдельного индекса.
При первом открытии хранилища происходит построение индекса с содержимым
файла и проверка целостности хранилища. Эта операция может занимать некоторое
время при открытии действительно больших файлов, поэтому данная библиотека
расчитана в первую очередь на небольшие хранилища.

По умолчанию все записи в хранилище заканчиваются вызовом метода
`os.File.Sync`, что позволяет быть до некоторой степени уверенными, что
данные при сбое не потеряются. Но, к сожалению, это одновременно сильно
замедляет любую операцию записи. Если вы хотите самостоятельно управлять
операциями сброса кешей файловой системы, то можно вызвать метод
`db.SetSync(false)` и затем вызывать метод `db.Sync` вручную из кода.

При удалении или перезаписи значений, свободные участки помечаются
специальным образом и в дальнейшем используются повторно, когда в них может
уместиться новая запись. Таким образом файл не очень сильно разрастается
при большом количестве удалений/вставок новый записей, при условии, что новые
значения не превышают по объему удаленные.

При работе с хранилищем поддерживаются групповые операции: `db.Gets`,
`db.Puts`, `db.Deletes`. Так же добавлены методы для работы с данными
в формате JSON и автоматического преобразования объектов в/из него.

Работа с ключами хранилища и выборки данных поддерживается единственным
методом `db.Keys`, который позволяет достаточно гибко выбирать только те
ключи, которые соответствуют заданным критериям. Список ключей всегда
возвращается в упорядоченном виде и может быть использован в дальнейшем
для выполнения групповых операций.

Для облегчения работы с данной библиотекой все методы хранилища
продублированы в виде глобальных функций, где первым параметром указывается
имя файла с хранилищем. Открытые таким образом хранилища кешируются в
глобальном списке и могут быть закрыты все сразу вызовом `CloseAll`.

```golang
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mdigger/keystore"
)

func main() {
	// автоматически закрыть по окончании все открытые хранилища
	defer keystore.CloseAll()
	var dbname = "test.db" // имя файла с хранилищем данных
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
	keys, err := keystore.Keys(dbname, "t", "", 0, 0, true)
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
}
```

### Примеры выборок ключей

- выбираем все ключи, которые начинаются на `test`

    	db.Keys("test", "", 0, 0, true)

- выбираем все ключи, которые начинаются на `test`, но после ключа `test2`

    	db.Keys("test", "test2", 0, 0, true)

- сортируем вывод в обратном порядке

    	db.Keys("", "", 0, 0, false)

- выбираем не более двух ключей

	    db.Keys("test", "test2", 0, 2, true)

- не используем префикс ключа, а выбираем по всем

	    db.Keys("", "test3", 0, 0, false)
