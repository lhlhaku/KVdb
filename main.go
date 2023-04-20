package KVdb

import (
	"encoding/binary"
	"fmt"
	"path"
	"strconv"
)

func main() {
	u16 := 1234
	u64 := 0x1020304040302010
	sbuf := make([]byte, 4)
	buf := make([]byte, 10)

	ret := binary.PutUvarint(sbuf, uint64(u16))
	fmt.Println(ret, len(strconv.Itoa(u16)), sbuf)

	ret = binary.PutUvarint(buf, uint64(u64))
	fmt.Println(path.Clean("/home/file//abc///aa.jpg"))
}
