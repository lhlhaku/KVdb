package index

import (
	"KVdb/data"
	"github.com/google/btree"
	"sync"
)

// BTree 索引，主要封装了 google 的 btree ku
// https://github.com/google/btree
type BTree struct {
	tree *btree.BTree  //读是并发安全的，写是并发并不安全的。所以需要加锁
	lock *sync.RWMutex //加锁
}

func NewBTree() *BTree {
	return &BTree{tree: btree.New(32),
		lock: new(sync.RWMutex)} ///控制叶子节点的数量
}

func (bt *BTree) Put(key []byte, pos *data.LogRecordPos) bool {
	it := &Item{key: key, pos: pos}
	bt.lock.Lock()
	bt.tree.ReplaceOrInsert(it)
	bt.lock.Unlock()
	return true

}
func (bt *BTree) Get(key []byte) *data.LogRecordPos {

	it := &Item{key: key}
	btreeItem := bt.tree.Get(it)
	if btreeItem == nil {
		return nil
	}
	return btreeItem.(*Item).pos

}
func (bt *BTree) Delete(key []byte) bool {

	it := &Item{key: key}
	bt.lock.Lock()
	oldItem := bt.tree.Delete(it)
	bt.lock.Unlock()
	if oldItem == nil {
		return false
	}
	return true

}
