# KV项目介绍

### 为什么用Go写：

1.go 有google开源的很好的btree的包，不用自己去实现了

2.go 单元测试非常方便 测试文件以_test.go结尾。

### KV存储

KV存储就是 键值数据存储，是一种基于键值对的存储方式，key为唯一标识符，value是与Key关联的数据。

目前最出名的KV存储产品就是Redis了，Redis是基于内存的kv数据库，它有一些持久化策略AOF和RDB，但是本质上还是基于内存设计的，数据的持久性不能得到完全保证。

我这里所说的KV存储是面向磁盘的，虽然在性能上无法和Redis匹敌，数据的持久性得到保证。

最关键的是Redis的数据受限于内存，而基于磁盘的KV存储，可以处理远超内存容量的数据，并且在性能上依然可以很强悍，这就是为什么KV存储价值巨大，也非常的受欢迎。

一般来说KV数据库的数据组织存储模型大致分为了两种，一个是B+树，一个是LSM树，基于B+树的项目比较著名的有BoltDB,而充分利用顺序IO、写性能更优的LSM Tree存储模型在近些年更加的受欢迎，其中最具有代表性的项目有LevelDB、RocksDB。



### bitcask模型介绍

一个bitcask实例就是系统上的一一个目录，并且限制同一时刻只能有一个进程打开这个目录。目录中有多个文件,同一时刻只有一个活跃的文件用于写入新的数据。当活跃文件写到满足一个阈值之后,就会被关闭，成为旧的数据文件,并且打开一个新的文件用于写入。所以这个目录中就是一个活跃文件和多个旧的数据文件的集合。



![image-20230420205911192](images\image-20230420205911192.png)

当前活跃文件的**写入是追加的**(append only)，这意味着可以利用**顺序IO**,**不会有多余的磁盘寻址,减少了磁盘寻道时间，最大限度保证了吞吐**。

![image-20230420210121326](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230420210121326.png)

每次写入都是追加写到活跃文件当中，**删除操作**实际上也是一次追加写入，只不过**写入的是一个特殊的墓碑值**，**用于标记一条记录的删除**, 也就是说不会实际去原地删除某条数据。

当下次merge的时候，才会将这种无效的数据清理掉。所以一个文件中的数据， 实际上就是多个相同格式的数据集合的排列：

![image-20230420210704699](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230420210704699.png)

**在追加写入磁盘文件完成后，然后更新内存中的数据结构，叫做keydir,实际上就是全部key的一个集合,存储的是key到一条磁盘文件数据的位置。**

**论文中说的是使用一个哈希表来存储，实际上这里的选择比较灵活，选用任意内存中的数据结构都是可以的,可以根据自己的需求来设计。**

例如哈希表，可以更高效的获取数据，但是无法遍历数据，如果想要数据有序遍历，可以选择B树、跳表等天然支持排序的数据结构。

![image-20230420210926737](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230420210926737.png)

keydir 一定会存储一条数据在磁盘中最新的位置，旧的数据仍然存在，等待merge的时候被清理。所以读取数据就会变得很简单，首先根据key从内存中找到对应的记录，这个记录存储的是数据在磁盘中的位置，然后根据这个位置，找到磁盘上对应的数据文件,以及文件中的具体偏移，这样就能够获取到完整的数据了。

![image-20230420211153046](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230420211153046.png)

由于旧的数据实际上一直存在于磁盘文件中，因为我们并没有将旧的数据删掉，而是新追加了一条标识其被删除的记录。所以随着bitcask 存储的数据越来越多，旧的数据也可能会越来越多。论文中提出了一个merge的过程来清理所有无效的数据。merge会遍历所有不可变的旧数据文件，将所有有效的数据重新写到新的数据文件中，且将旧的数据文件删除掉。

![image-20230420211518906](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230420211518906.png)

merge完成后，还会为每个数据文件生成一个hint文件，hint 文件可以看做是全部数据的索引，它和数据文件唯一的区别是，它不会存储实际的value。它的作用是在bitcask 启动的时候，直接加载hint文件中的数据，快速构建索引，而不用去全部重新加载数据文件，换句话说，就是在启动的时候加载更少的数据，因为hint文件不存储value,它的容量会比数据文件小。

bitcask的设计特点：

![image-20230420212042958](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230420212042958.png)

### 内存设计

首先是内存，在内存当中，我们需要一种支持高效插入、读取、删除数据的结构，并且如果需要数据高效遍历的话，我们最好是选择天然支持有序的一种结构。所以说常见的选择BTree、 跳表、红黑树等。

**选择常用的BTree结构，使用google 的Github Repo下开源了一个BTree的库**

### 磁盘设计

目前只支持标准的系统文件IO，将标准文件操作API例如read、write、 close 等方法进行简单的封装。

![image-20230421185509302](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230421185509302.png)

### 刚启动db

##### 读取数据文件：

读取传入文件数据文件目录，读取目录中所有以.data的文件（001.data、002.data）并按照id排序。然后以此打开文件，将文件操作符记录到db.olderFiles和db.activeFile。其中db.activeFile是id最大的文件。

##### 加载索引：

遍历所有文件，依次读取文件内容，根据提供的offset 自动读取文件中一条记录，并返回这条数据的长度大小。生成 key - (fileid,offset)写入到索引中，更新offset，继续读取。

如果读取的是活跃文件，会自动维护一个offset，方便重启完成后，新写入数据。



### 如何保证重启db时内存索引数据是有效最新的

##### 删除的情况

先写入后，后删除，此时数据文件里有两条数据，一前一后。我们在重启db时，从头遍历数据文件。读入（写入的一行记录），会向索引中添加 key - (fileid,offset)。然后再读入删除的操作时，也就是（一条记录里type=deleted），依旧会写入。

我们在merge操作时，不仅判断数据（type=deleted）无效，**并删除此时索引中的key**（不管有没有）

后续有多余的旧文件就需要Merge操作

##### 修改的情况

由于读数据文件是从头读的，当第二次读入相同key不同value时，新生成的key - (fileid,offset)会覆盖掉旧的key - (fileid,offset)，也就是内存索引会发生覆盖。

但是会造成数据文件中旧数据过多，所以需要Merge操作



### 删除数据

首先会判断key是否有效和存在，因为我们是先写入到数据文件然后更新内存索引，如果不判断的话，就一直写入到数据文件，而且这个数据文件是无效的。

删除流程：添加一条数据记录，数据记录的墓碑值置为true，写入到数据文件，然后删除内存索引。



### 数据文件的记录格式

数据文件中，一条记录的格式：

![image-20230423155425787](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230423155425787.png)





### 写入数据文件细节流程：

```go
// 构造 LogRecord 结构体
logRecord := &data.LogRecord{
   Key:   logRecordKeyWithSeq(key, nonTransactionSeqNo),
   Value: value,
   Type:  data.LogRecordNormal,
}
```

就是先构造一个logRecord，然后获取活跃文件的操作符，然后判断活跃文件是否为空，为空则创建一个活跃文件。然后对logRecord进行编码：

![image-20230423162508321](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230423162508321.png)

上面是一条存储在实际数据文件中的一条加密后的logRecord记录。

头部head结构：

```go
type logRecordHeader struct {
   crc        uint32        // crc 校验值
   recordType LogRecordType // 标识 LogRecord 的类型
   keySize    uint32        // key 的长度
   valueSize  uint32        // value 的长度
}
const maxLogRecordHeaderSize = binary.MaxVarintLen32*2 + 5 //头部最大长度
// crc type keySize valueSize
// 4 +  1  +  5   +   5 = 15
```

这里结构体定义keySize，valueSize有点问题

所以头部是变长的，4+1+（5）+（5）也就是最大长度为15字节 

具体加密过程如下图所示：

对头部进行加密：头部的Key_size,Value_size 是边长加密的

然后对整个encBytes进行crc校验，并放到encBytes头部

```go
func EncodeLogRecord(logRecord *LogRecord) ([]byte, int64) {
   // 初始化一个 header 部分的字节数组
   header := make([]byte, maxLogRecordHeaderSize)

   // 第五个字节存储 Type
   header[4] = logRecord.Type
   var index = 5
   // 5 字节之后，存储的是 key 和 value 的长度信息
   // 使用变长类型，节省空间
   index += binary.PutVarint(header[index:], int64(len(logRecord.Key)))
   index += binary.PutVarint(header[index:], int64(len(logRecord.Value)))

   var size = index + len(logRecord.Key) + len(logRecord.Value)
   encBytes := make([]byte, size)

   // 将 header 部分的内容拷贝过来
   copy(encBytes[:index], header[:index])
   // 将 key 和 value 数据拷贝到字节数组中
   copy(encBytes[index:], logRecord.Key)
   copy(encBytes[index+len(logRecord.Key):], logRecord.Value)

   // 对整个 LogRecord 的数据进行 crc 校验
   crc := crc32.ChecksumIEEE(encBytes[4:])
   binary.LittleEndian.PutUint32(encBytes[:4], crc)

   return encBytes, int64(size)
}
```



之后得到加密校验后的encBytes写入到文件中，写入之前需要判断超过文件的最大长度。

### 获取数据记录的细节流程：

```go
// LogRecordPos 数据内存索引，主要是描述数据在磁盘上的位置
type LogRecordPos struct {
   Fid    uint32 // 文件 id，表示将数据存储到了哪个文件当中
   Offset int64  // 偏移，表示将数据存储到了数据文件中的哪个位置
}
```

通过key得到LogRecordPos记录，里面有文件和文件里偏移位置

我们拿到offset去读数据记录时，先读的头部，但是我们不知道头部有多长，所以直接按照最大长度去读maxLogRecordHeaderSize，它的后面两个数据Key_size,Value_size是加密变长的，就算我们按照最大长度去读，也可以通过解密得到实际值。

拿到header后解密：

```go
// 对字节数组中的 Header 信息进行解码
func decodeLogRecordHeader(buf []byte) (*logRecordHeader, int64) {
   if len(buf) <= 4 {
      return nil, 0
   }

   header := &logRecordHeader{
      crc:        binary.LittleEndian.Uint32(buf[:4]),
      recordType: buf[4],
   }

   var index = 5
   // 取出实际的 key size
   keySize, n := binary.Varint(buf[index:])
   header.keySize = uint32(keySize)
   index += n

   // 取出实际的 value size
   valueSize, n := binary.Varint(buf[index:])
   header.valueSize = uint32(valueSize)
   index += n

   return header, int64(index)
}
```

拿到一个完整的头部，也就意味着拿到Key_size,Value_size，之后再往后读响应的长度，就得到了key，value。然后将header里的crc和整个header+key+value进行校验。校验通过返回LogRecord。

总之，存储在数据文件中的logrecord和我们在项目中传递的logrecord不一样。

文件中：header+key+value，项目中：key+value+type。

##### 一个小bug：

我们在读取头部时，是按照最大长度去读的，如果一个文件末尾，恰好有一条记录，这个记录header+key+value都没有maxLogRecordHeaderSize长，所以会报错，所以需要判断一下：

```go
// 如果读取的最大 header 长度已经超过了文件的长度，则只需要读取到文件的末尾即可
var headerBytes int64 = maxLogRecordHeaderSize
if offset+maxLogRecordHeaderSize > fileSize {
   headerBytes = fileSize - offset
}
```

##### 大端序小端序：

![image-20230423171746234](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230423171746234.png)

### 构建索引迭代器

这个迭代器是给用户用的

```go
// Iterator 通用索引迭代器
type Iterator interface {
   // Rewind 重新回到迭代器的起点，即第一个数据
   Rewind()
   // Seek 根据传入的 key 查找到第一个大于（或小于）等于的目标 key，根据从这个 key 开始遍历
   Seek(key []byte)
   // Next 跳转到下一个 key
   Next()
   // Valid 是否有效，即是否已经遍历完了所有的 key，用于退出遍历
   Valid() bool
   // Key 当前遍历位置的 Key 数据
   Key() []byte
   // Value 当前遍历位置的 Value 数据
   Value() *data.LogRecordPos
   // Close 关闭迭代器，释放相应资源
   Close()
}
```

```go
func newBTreeIterator(tree *btree.BTree, reverse bool) *btreeIterator {
   var idx int
   values := make([]*Item, tree.Len())

   // 将所有的数据存放到数组中
   saveValues := func(it btree.Item) bool {
      values[idx] = it.(*Item)
      idx++
      return true
   }
   if reverse {
      tree.Descend(saveValues)
   } else {
      tree.Ascend(saveValues)
   }

   return &btreeIterator{
      currIndex: 0,
      reverse:   reverse,
      values:    values,
   }
}
```

这里有个致命的缺点是：我们需要把所有的key和LogRecordPos一起保存到数组中，这会导致内存的膨胀，但是这是无法避免的，因为我们使用的btree索引，无法实现用户期望的索引迭代功能。所以必须取保存所有的key和LogRecordPos去自定义迭代索引方法。

也就是每创建一个迭代器都需要在内存保存所有的key和LogRecordPos

后续如果有其他索引的数据类型，再说，目前btree索引必须要这么做。

还有一个缺点是：如果我们新建一个索引迭代器后，后续有新的key写入，这个迭代器是无法得到查询到的。

```go
type Item struct {
   key []byte
   pos *data.LogRecordPos
}
```

创建逻辑：

创建一个db实例，然后调用new Iterator 创建迭代器，主要是通过db.index去创建一个索引迭代器



### 关闭数据库实例流程

```go
// Close 关闭数据库   //关闭当前的活跃文件
func (db *DB) Close() error {

   if db.activeFile == nil {
      return nil
   }
   db.mu.Lock()
   defer db.mu.Unlock()

   // 关闭索引
   if err := db.index.Close(); err != nil {
      return err
   }

   // 关闭当前活跃文件
   if err := db.activeFile.Close(); err != nil {
      return err
   }
   // 关闭旧的数据文件
   for _, file := range db.olderFiles {
      if err := file.Close(); err != nil {
         return err
      }
   }
   return nil
}
```

依次关闭索引和数据文件

### WriteBatch原子写

我们的bitcask存储引擎设计中，有个很大的特点就是：我们需要把所有的key维护到内存中，如果在此基础上实现mvcc，那么也会在内存中去维护所有的key、位置索引、版本信息，那么这可能会造成内存容量的急剧膨胀。

本次设计的是简单的事务，利用全局锁保证串行化。也就是批量写入，事务在没有提交前，之前事务中的所有操作不可见， 并且，如果执行事务异常，没有提交，已经完成的操作是会回退。

```go
// WriteBatch 原子批量写数据，保证原子性
type WriteBatch struct {
   options       WriteBatchOptions
   mu            *sync.Mutex
   db            *DB
   pendingWrites map[string]*data.LogRecord // 暂存用户写入的数据
}
```

```go
// Commit 提交事务，将暂存的数据写到数据文件，并更新内存索引
func (wb *WriteBatch) Commit() error {
   wb.mu.Lock()
   defer wb.mu.Unlock()

   if len(wb.pendingWrites) == 0 {
      return nil
   }
   if uint(len(wb.pendingWrites)) > wb.options.MaxBatchNum {
      return ErrExceedMaxBatchNum
   }

   // 加锁保证事务提交串行化
   wb.db.mu.Lock()
   defer wb.db.mu.Unlock()

   // 获取当前最新的事务序列号
   seqNo := atomic.AddUint64(&wb.db.seqNo, 1)

   // 开始写数据到数据文件当中
   positions := make(map[string]*data.LogRecordPos)
   for _, record := range wb.pendingWrites {
      logRecordPos, err := wb.db.appendLogRecord(&data.LogRecord{
         Key:   logRecordKeyWithSeq(record.Key, seqNo),
         Value: record.Value,
         Type:  record.Type,
      })
      if err != nil {
         return err
      }
      positions[string(record.Key)] = logRecordPos
   }

   // 写一条标识事务完成的数据
   finishedRecord := &data.LogRecord{
      Key:  logRecordKeyWithSeq(txnFinKey, seqNo),
      Type: data.LogRecordTxnFinished,
   }
   if _, err := wb.db.appendLogRecord(finishedRecord); err != nil {
      return err
   }

   // 根据配置决定是否持久化
   if wb.options.SyncWrites && wb.db.activeFile != nil {
      if err := wb.db.activeFile.Sync(); err != nil {
         return err
      }
   }

   // 更新内存索引
   for _, record := range wb.pendingWrites {
      pos := positions[string(record.Key)]
      if record.Type == data.LogRecordNormal {
         wb.db.index.Put(record.Key, pos)
      }
      if record.Type == data.LogRecordDeleted {
         wb.db.index.Delete(record.Key)
      }
   }

   // 清空暂存数据
   wb.pendingWrites = make(map[string]*data.LogRecord)

   return nil
}
```

事务提交时，需要保证串行化，会加锁。

而且会对事务操作的里提交的数据的key和事务id进行联合加密：

logRecordKeyWithSeq(txnFinKey, seqNo)

也就是对事务id进行加密，然后拼接到key的头部：

```go
// key+Seq Number 编码
func logRecordKeyWithSeq(key []byte, seqNo uint64) []byte {
   seq := make([]byte, binary.MaxVarintLen64)
   n := binary.PutUvarint(seq[:], seqNo)

   encKey := make([]byte, n+len(key))
   copy(encKey[:n], seq[:n])
   copy(encKey[n:], key)

   return encKey
}
```

我们在写入一条记录时，传给数据文件层只需要一个

```go
// LogRecord 写入到数据文件的记录
// 之所以叫日志，是因为数据文件中的数据是追加写入的，类似日志的格式
type LogRecord struct {
   Key   []byte
   Value []byte
   Type  LogRecordType
}
```

文件层会帮我们编码，写入到数据文件中。

**把事务中所有记录保存到数据文件中后，还需要往数据文件中，添加一条特殊的记录**：

```Go
// 写一条标识事务完成的数据
finishedRecord := &data.LogRecord{
   Key:  logRecordKeyWithSeq(txnFinKey, seqNo),
   Type: data.LogRecordTxnFinished,
}
if _, err := wb.db.appendLogRecord(finishedRecord); err != nil {
   return err
}
```

最后再更新内存，注意那条特殊的记录不需要更新到内存。



### db初始化时WriteBatach原子写初始化

会加载所有数据文件的记录然后生成一条索引。

其中需要注意的是，有些事务没有完成，但是记录已经写进去了，虽然这个记录是无效的，所以需要过滤。



此时会对前面的操作重新设计，把每条记录分为：

默认事务id:nonTransactionSeqNo（非事务操作），事务id(事务操作)

读取一条记录，如果是事务id:nonTransactionSeqNo

```go
logRecordPos := &data.LogRecordPos{Fid: fileId, Offset: offset}

// 解析 key，拿到事务序列号
realKey, seqNo := parseLogRecordKey(logRecord.Key)
if seqNo == nonTransactionSeqNo {
   // 非事务操作，直接更新内存索引
   updateIndex(realKey, logRecord.Type, logRecordPos)
}
```

如果有事务id:

```go
// 事务完成，对应的 seq no 的数据可以更新到内存索引中
if logRecord.Type == data.LogRecordTxnFinished {
   for _, txnRecord := range transactionRecords[seqNo] {
      updateIndex(txnRecord.Record.Key, txnRecord.Record.Type, txnRecord.Pos)
   }
   delete(transactionRecords, seqNo)
} else {
   logRecord.Key = realKey
   transactionRecords[seqNo] = append(transactionRecords[seqNo], &data.TransactionRecord{
      Record: logRecord,
      Pos:    logRecordPos,
   })
}
```

如果没有读到事务完成的那条特殊记录，会把记录按照事务id保存到map中，等到读到特殊记录，把那条特殊记录的事务id的map保存到索引中。



### Merge操作

##### 无效数据来源

1. 相同的key不同的value进行重写
2. 增加一条墓碑值为true的记录，原来的记录无效
3. writebatch事务无效操作，有些无效数据已经写入到磁盘中

##### Merge设计目标

bitcask论文中对Merge只是做了简单的描述，但是怎么去实现还需要我们仔细斟酌。

Merge 的主要设计目标有两个点，一是清理旧的数据，重写有效的数据，二是生成只包含索引的hint文件,并且我们需要尽量保证这个过程不对前台正常的读写造成太大的影响。

hint文件的作用：保存一些merge后的有效索引，加快启动速度，不需要再去重复的读数据文件生成索引。

##### 清理无效数据

对于第一点，将磁盘上的无效数据清理掉，具体的做法其实有很多种。我们可以**按编号从小到大读取数据文件,并且依次取出其中的每一条日志记录，然后跟内存中的索引进行比较**，如果和内存索引对应的文件id和偏移offset一致， 说明这是有效的数据，然后直接调用Put接口重写这条数据即可，一个文件中的数据重写完了，就将其删除掉。
**但是这样会使用面向外层调用者的Put接口，增加这个方法的锁竞争。**

这样做还有一个不太好处理的点，**就是如果一个文件中的数据重写到了一 半,但是出错了，这时候新的数据文件中有新重写的数据，而旧的文件又不能删除掉**。**解决这个问题的一个方法是将一个文件中的数据全部放到一个事务中执行,只有当事务提交成功之后,才能够删除文件**。但是如果一个文件中的数据太多的话，提交事务之前，**一般会将数据批量缓存到内存中，这样可能会造成内存容量的膨胀**。

（因为我们内存中已经保存所有的key了，再加上事务保存的临时重写key）很容易就超内存。

所以我们可以换一种思路,使用一个临时的文件夹，**假如叫merge,在这个临时目录中新启动一个数据库实例，这个实例和正在运行的数据库实例互不冲突，因为它们是不同的进程。**将原来的目录中的数据文件逐一读取，并取出其中的日志记录，和内存索引进行比较,如果是有效的,我们将其重写到merge这个数据目录中，**避免和原来的目录竞争**。

避免了和用户层面的写操作竞争，尽可能降低对用户的影响。

![image-20230424201227104](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230424201227104.png)

### hint索引文件

重写完成后内存索引，key-logPOs。记录merger完成的key在新的Merge目录中的位置。

### 重启校验merge

上一次merge过程中，用户还在写入数据，或者说merge了一部分旧数据。所以，对于原始数据而言，此时有一部数据文件被merge了，有一部分数据文件没有Merge。

merge完成后的数据文件直接被替换，索引的加载加载hint索引文件。

有没有被merge的文件需要遍历数据文件建立索引。

不用害怕一种情况：一条记录被merge后写入到重写数据文件中，hint文件也记录了这条索引记录，之后用户有对这条记录进行修改或者删除。此时hint文件数据记录是过时的。

我们在加载索引时，先加载hint文件，然后遍历未merge文件生成索引，旧的无效的索引一定会被覆盖。



### merge操作具体流程

首先会加锁，把当前的活跃文件变成旧的文件，然后新建一个活跃文件，然后把所有的旧的文件保存到一个数组记录起来，然后释放锁。

所以，我们merge的文件不可能包含所有的文件，释放锁后，如果用户不断写入添加了很多的旧数据文件，本轮的merge也无法处理。

新建一个merge db 这也是一个bitcask实例

我们在判断数据是否有效时，只要他的这条记录在内存中，而且文件id和偏移量都一样说明是有效的，而且也可以忽略他的事务id了，因为已经出现在内存中，就说明是有效的

```Go
if db.activeFile == nil {
   return nil
}

db.mu.Lock()
if db.isMerging {
   db.mu.Unlock()
   return ErrMergeIsProgress
}
db.isMerging = true
defer func() {
   db.isMerging = false
}()

if err := db.activeFile.Sync(); err != nil {
   db.mu.Unlock()
   return err
}
//当前活跃文件变成旧的文件
db.olderFiles[db.activeFile.FileId] = db.activeFile

if err := db.setActiveDataFile(); err != nil {
   db.mu.Unlock()
   return nil
}
nonMergeFileId := db.activeFile.FileId

//创建一个用于保存所有需要merge的文件
var mergeFiles []*data.DataFile
for _, file := range db.olderFiles {
   mergeFiles = append(mergeFiles, file)
}
db.mu.Unlock()
```

```go
// 新建一个 merge path 的目录
if err := os.MkdirAll(mergePath, os.ModePerm); err != nil {
   return err
}
// 打开一个新的临时 bitcask 实例
mergeOptions := db.options
mergeOptions.DirPath = mergePath
mergeOptions.SyncWrites = false
mergeDB, err := Open(mergeOptions)
if err != nil {
   return err
}

// 打开 hint 文件存储索引
hintFile, err := data.OpenHintFile(mergePath)
if err != nil {
   return err
}
// 遍历处理每个数据文件
for _, dataFile := range mergeFiles {
   var offset int64 = 0
   for {
      logRecord, size, err := dataFile.ReadLogRecord(offset)
      if err != nil {
         if err == io.EOF {
            break
         }
         return err
      }
      // 解析拿到实际的 key
      realKey, _ := parseLogRecordKey(logRecord.Key)
      logRecordPos := db.index.Get(realKey)
      // 和内存中的索引位置进行比较，如果有效则重写
      if logRecordPos != nil &&
         logRecordPos.Fid == dataFile.FileId &&
         logRecordPos.Offset == offset {
         // 清除事务标记
         logRecord.Key = logRecordKeyWithSeq(realKey, nonTransactionSeqNo)
         pos, err := mergeDB.appendLogRecord(logRecord)
         if err != nil {
            return err
         }
         // 将当前位置索引写到 Hint 文件当中
         if err := hintFile.WriteHintRecord(realKey, pos); err != nil {
            return err
         }
      }
      // 增加 offset
      offset += size
   }
}
```

实现上：需要merge所有merge开始时的所有数据文件。

merge完成后，通过创建一个文件merge.finished，记录merge到哪的数据文件，里面有一条记录:

```go
mergeFinRecord := &data.LogRecord{
   Key:   []byte(mergeFinishedKey),
   Value: []byte(strconv.Itoa(int(nonMergeFileId))),
}
```

重启加载merge完成后替换数据文件，加载索引，会删除merge目录。怎样判断呢？凡是比nonMergeFileId小的文件说明就已经被merge了

```Go
// 将新的数据文件移动到数据目录中
for _, fileName := range mergeFileNames {
   srcPath := filepath.Join(mergePath, fileName)
   destPath := filepath.Join(db.options.DirPath, fileName)
   if err := os.Rename(srcPath, destPath); err != nil {
      return err
   }
}
return nil
```

是怎么移动文件的呢？首先把被merge的文件删除，然后把merge目录的文件移动过去。为什么可以移动，因为：merge目录文件一定比数据文件目录小，所以文件id一定会小于或者等于的数据文件目录未merge的文件id的，这样就避免冲突了。

这里的逻辑在从未merge文件中加载索引也用到了，我们判断所有数据文件是否小于nonMergeFileId，小于则不用再加载索引了，在loadIndexFromHintFiles（）方法阶段已经做过了。

##### 启动db的流程

```go
//查看是否已经被merge过了

if err := db.loadMergeFiles(); err != nil {
   return nil, err
}

// 加载数据文件
if err := db.loadDataFiles(); err != nil {
   return nil, err
}
//加载Hint文件构建已经被merge的数据记录索引

if err := db.loadIndexFromHintFiles(); err != nil {
   return nil, err

}

// 从数据文件中加载索引
if err := db.loadIndexFromDataFiles(); err != nil {
   return nil, err
}

return db, nil
```

### 使用的自适应基数树进行对索引优化

自适应基数树和B树的区别，以及和其他索引结构的区别

### 内存索引的优化

bitcask内存索引的好处：

从bitcask论文中可以得知，其实这个存储模型最大的特点是所有的索引都只能在内存中维护，这样的特性带来了一个**很大的好处**，那就是只需要从内存中就能够直接获取到数据的索引信息，然后只通过一次磁盘IO操作就可以拿到数据了。

**带来的问题：**

但是拥有这个好处的同时也带来了一个缺陷，那便是我们的存储引擎能维护多少索引，完全取决于内存容量，也就是说**数据库能存储的key索引的数据量受到了内存容量的限制**。

**解决问题的办法有很多**：规避这个问题的方案其实有很多，**我们可以选择一个节省内存空间的数据结构作为索引**，或者直接将索引存储到磁盘当中例如**使用持久化的B+树作为索引**。

如：

![image-20230427203920157](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230427203920157.png)

当然天下没有免费的午餐，如果将索引存储到了磁盘当中，好处是可以节省内存空间，突破存储引擎的数据量受内存容量的限制，但是随之而来的缺点也很明显，那就是读写性能会随之降低,因为需要从磁盘上获取索引，然后再去磁盘数据文件中获取value。

### 使用B+树将索引保存到磁盘，突破内存限制索引的情况

使用开源的boltdb的库，这个库是一个标准的B+树实现，也是go生态中较为知名的一个KV库。

```go
// BPlusTree B+ 树索引
// 主要封装了 go.etcd.io/bbolt 库
type BPlusTree struct {
   tree *bbolt.DB
}
```

这里没有加锁的原因就是：boltdb内部支持并发访问，所以这里不需要加锁了。

```go
// NewBPlusTree 初始化 B+ 树索引
func NewBPlusTree(dirPath string, syncWrites bool) *BPlusTree {
   opts := bbolt.DefaultOptions
   opts.NoSync = !syncWrites
   bptree, err := bbolt.Open(filepath.Join(dirPath, bptreeIndexFileName), 0644, opts)
   if err != nil {
      panic("failed to open bptree")
   }

   // 创建对应的 bucket
   if err := bptree.Update(func(tx *bbolt.Tx) error {
      _, err := tx.CreateBucketIfNotExists(indexBucketName)
      return err
   }); err != nil {
      panic("failed to create bucket in bptree")
   }

   return &BPlusTree{tree: bptree}
}
```

**一个索引文件由一个bucket所代表**

因为boltdb本身就是一个db实例，索引保存在磁盘文件中，所以我们需要指定一个目录作为保存索引的地址。

由update内部函数可知：Update内部把开启一个事务去执行我们在update里执行的方法，如果执行成功，事务提交，执行失败，则事务回滚。

view函数和update类似，主要是view 不可以写，update可以写

##### B+树的迭代器

使用的也是boltdb内部实现好的迭代器

```go
// B+树迭代器
type bptreeIterator struct {
   tx        *bbolt.Tx     //事务
   cursor    *bbolt.Cursor //光标，相当于一个迭代器去遍历bucket
   reverse   bool
   currKey   []byte
   currValue []byte
}
```

**B+树为索引的db在初始化时，不需要从数据文件中加载索引**

![image-20230427215617606](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230427215617606.png)

### 采用B+树对系统产生的变化：

![image-20230427215515317](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230427215515317.png)

需要保存当前的事务序列号到文件中，apt树和b树获取事务号可以从数据中获得事务号，但是b+树由于不需要扫描数据文件，直接从索引文件中获取索引，所以就无法得到事务号。所以需要保存当前的事务号到文件中。



如果当前索引是b+树并且没有那个保存事务的文件，则我们直接禁用掉writebatch的功能。但是有一个情况，当用户第一次使用时，是没有那个文件的，所以我们还需要一个标记量。

下面的三个判断条件：

![image-20230429202824213](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230429202824213.png)

![image-20230429203328007](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230429203328007.png)

所以我们又多了两种记录文件：记录Merge完成文件id，记录最新的事务号。



### 文件锁

为了保证设计的简洁性，只允许存储引擎实例在单进程执行，如何保证呢？

目前的Open打开存储引擎的方法并没有提供这个保证。我们可以加上一个文件锁，文件锁是一种进程间通信的常用方式，有一个系统调用flock提供了文件锁的功能，主要保证了多个进程之间的互斥，恰好能够满足我们的需求。对于Go语言，可以使用这个现成的库https://github.com/gofrs/flock,其底层主要调用了Flock。也可以自己实现，这里直接调用轮子了。

创建数据库实例：

```go
// 判断当前数据目录是否正在使用
fileLock := flock.New(filepath.Join(options.DirPath, fileLockName))
//尝试去获取这把锁
hold, err := fileLock.TryLock()
if err != nil {
   return nil, err
}
if !hold {
   return nil, ErrDatabaseIsUsing
}
```

关闭数据库时，关闭锁：

![image-20230430185345916](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230430185345916.png)



### 持久化策略优化

![image-20230430190337289](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230430190337289.png)

之前提供了一个Sync的选项，主要是控制是否每条数据都持久化到磁盘，目前的默认值是false, 也就是将持久化这个行为交给了操作系统进行调度。
但是在实际环境中，我们可以再提供另一个选项,用户可以设置积累到多少个字节之后，再进行一次持久化。这相当于提供了一个折中的方案，相较于之前的要么每条都持久化，要么完全交给操作系统，这样能够让用户自己灵活配
置。|
具体的做法也比较简单，可以在打开数据库的时候，增加一个配置项BytesPerSync，每次写数据的时候，都记录一下累计写了多少个字节，如果累计值达到了BytesPerSync,则进行持久化。

### 启动速度优化

![image-20230430190641867](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230430190641867.png)

mmap实现iomanager接口

（在index不是b+树的情况）。bitcask 在启动的时候，会全量加载所有的数据并构建内存索引，在数据量较大的情况下，这个构建的过
程可能会非常漫长。这带来的问题是重启时间过长，例如数据库崩溃之后需要立即启动，并马上接受用户请求，这时候用户等待的时间就会比较长。

在之前的默认文件I0下，会涉及到和内核进行交互，执行文件系统调用接口，操作系统会将内核态的数据拷贝到用户态。

所以我们可以使用**内存文件映射**(MMap) I0 来进行启动的加速，mmap指的是将磁盘文件映射到内存地址空间，操作系统在读文件的时候，会触发缺页中断，将数据加载到内存中，这个内存是用户可见的，相较于传统的文件I0,避免了从内核态到用户态的数据拷贝。

**内存文件映射是指将一个文件映射到进程的地址空间，使得进程可以像访问内存一样访问文件，而不需要进行繁琐的read和write操作。内存文件映射是一种高效的I/O操作方式，它可以减少系统调用次数，提高I/O效率**

我们可以提供一个配置项,让用户决定是否在启动的时候使用MMap，如果是的话，则打开数据文件的时候，我们将按照MMap l0的方式初始化l0Manager,加载索引时读数据都会使用mmap。加载索引完成后，我们需要重置我们的I0Manager,因为MMap只是用于数据库启动，启动完成之后，要将IOManager切换到原来的I0类型。

**也就是说MMap用来加速数据库启动流程，也就是读操作，写操作可不可以呢？**

**因为：普通io不需要在内存开辟空间映射文件全部内容，只需要把写入内容写入内存用户态缓冲区-》内核缓冲区-》磁盘文件。虽然多了一次copy，但是比mmio节省内存**

##### MMAP原理：

内存有内核缓冲区和用户缓冲区，普通io会发生两次copy，mmap一次copy,不涉及内核缓冲区

[一文搞懂内存映射(Memory Map)原理 - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/473643975)

[(8条消息) 操作系统-IO与零拷贝【万字文，比较详细的解析】_零拷贝和异步io的区别_youthlql的博客-CSDN博客](https://blog.csdn.net/Youth_lql/article/details/115524139)

##### 内核态用户态：

打开磁盘文件并读取到内存需要把数据从内核态拷贝到用户态。这个过程涉及到用户态和内核态的切换，因此会有一定的性能损耗。

内核态和用户态是操作系统的两种运行级别。当程序运行在3级特权级上时，就可以称之为运行在用户态。因为这是最低特权级，是普通的用户进程运行的特权级，大部分用户直接面对的程序都是运行在用户态。当程序运行在0级特权级上时，就可以称之为运行在内核态。**运行在用户态下的程序不能直接访问操作系统内核数据结构和程序。当我们在系统中执行一个程序时，大部分时间是运行在用户态下的，在其需要操作系统帮助完成某些它没有权力和能力完成的工作时就会切换到内核态**（比如操作硬件）

特权级是指CPU在执行指令时的权限等级，通常分为0、1、2、3级别，其中0级别最高，3级别最低。操作系统位于0级特权，可以直接控制硬件，掌控各种核心数据；系统程序分别位于1级特权或2级特权，主要是一些虚拟机、驱动程序等系统服务；而一般的应用程序运行在3级特权。当CPU需要访问更高特权级的计算机资源时，CPU就需要进行特权级切换，使得CPU处于更高的特权级下，内核被赋予0级特权



### 数据merge优化

基础的merge流程是挨个遍历数据文件进行回收清理，但如果我们的存储引擎中，无效的数据本身就很少(或者没有无效的数据)， 那么全量的遍历整个数据文件,然后依次重写有效数据的操作代价较高，可能会导致严重的磁盘空间和带宽浪费。
所以我们可以在存储弓|擎运行的过程当中统计有多少数据量是有效的,这样会得到一个实时的失效数据总量，再让户决定是否进行merge操作。



那么应该如何统计失效的数据量呢?我们可以在内存索引中维护一个值，记录每条数据在磁盘 上的大小。

**Delete 数据的时候，可以得到旧的值，这个旧的值就是磁盘上失效的数据。**
**Put存数据的时候，如果判断到有旧的数据存在，那么也同样累加这个值。**



btree和art树put,delete都会返回旧值，直接用就行。

bptree需要提前调用get旧值

![image-20230430214136565](C:\Users\Haku\AppData\Roaming\Typora\typora-user-images\image-20230430214136565.png)

delete本身新加的数据也是需要清理的

这样我们就能够从Put/Delete数据的流程中，得到失效数据的累计值。

这里需要改动我们之前的索引数据结构的返回值，put 的时候，将之前的I旧值返回出来，delete 的时候，将值返回出来，然后使用的时候，我们拿到这个值如果不为空的话，就增加累计值。

我们可以提供一个配置项,只有当失效的数据占比到达了某个比例之后，才进行Merge操作。
**这里我们可以提供一个Stat 的方法**，返回存储引擎的一些统计信息，包含目前失效的数据量，当然我们可以加上其他的属性，比如数据库中Key的数量、数据文件的个数、占据磁盘总空间等。



##### 磁盘空间判断

merge的时候，我们开启了一个临时目录,并且将有效的数据全部存放到这个临时目录中，但是设想这样一个极端情况:如果原本的数据目录本身就很大了，并且数据库中失效的数据很少(或者根本没有失效数据)，那么在merge完成后，磁盘上有可能存在两倍于原始数据容量的数据，这有可能会导致磁盘空间被写满。
所以我们可以在merge之前,加上一个判断，查看当前数据目录所在磁盘，是否有足够的空间容纳Merge后的数据,避免由于数据量太大导致磁盘空间被写满。
**具体的做法也很简单，我们添加一个获取目录所占容量的方法，然后再获取到所在磁盘的剩余容量，如果merge后的数据量超过了磁盘的剩余容量，那么直接返回一个磁盘空间不足的错误信息。**

