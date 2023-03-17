package main

import (
	kvdb "KVdb"
	"fmt"
)

func main() {

	opts := kvdb.DefaultOptions
	opts.DirPath = "/tmp/KVdb"

	db, err := kvdb.Open(opts)

	if err != nil {
		panic(err)
	}

	err = db.Put([]byte("name"), []byte("bitcasK"))
	if err != nil {
		panic(err)
	}

	val, err := db.Get([]byte("name"))
	if err != nil {
		panic(err)
	}
	fmt.Println("val=", string(val))
	err = db.Delete([]byte("name"))

	if err != nil {
		panic(err)
	}

}
