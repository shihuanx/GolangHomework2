package dao

import (
	"container/list"
	"log"
	"math/rand"
	"sync"
	"time"
)

var Expiration = time.Hour

// MemoryDBDao 定义内存数据库结构体
type MemoryDBDao struct {
	dataMap    map[string]interface{}
	expires    map[string]time.Time
	rwLock     sync.RWMutex
	capacity   int                      // 最大内存容量（键值对数量）
	lruList    *list.List               // 双向链表，用于实现 LRU 内存淘汰
	lruMap     map[string]*list.Element //键和链表元素的映射map
	evictRatio float64                  // 淘汰比例
}

// NewMemoryDBDao 初始化内存数据库实例
func NewMemoryDBDao(capacity int, evictRatio float64) *MemoryDBDao {
	mdb := &MemoryDBDao{
		dataMap:    make(map[string]interface{}),
		expires:    make(map[string]time.Time),
		capacity:   capacity,
		lruList:    list.New(),
		lruMap:     make(map[string]*list.Element),
		evictRatio: evictRatio,
	}
	return mdb
}

// Set 设置键值对并设置过期时间
func (mdb *MemoryDBDao) Set(key string, value interface{}, expiration int64) {
	nanoseconds := expiration * int64(time.Second)
	duration := time.Duration(nanoseconds)
	mdb.rwLock.Lock()
	defer mdb.rwLock.Unlock()

	//只在添加键时断内存满没满 满了就执行内存淘汰 再添加键 其余操作只把键添加到链表头 然后淘汰从链表尾选取一部分淘汰
	if len(mdb.dataMap) >= mdb.capacity {
		mdb.evict()
	}
	element := mdb.lruList.PushFront(key)
	mdb.lruMap[key] = element

	// 如果过期时间大于 0 就设置过期时间，如果过期时间为 0 说明这个键永不过期
	if expiration > 0 {
		mdb.expires[key] = time.Now().Add(duration)
		mdb.dataMap[key] = value
		log.Printf("已添加键：%s 值：%v 过期时间：%v", key, value, mdb.expires[key])
	} else {
		mdb.dataMap[key] = value
		log.Printf("已添加键：%s 值：%v", key, value)
	}
}

// Get 获取键对应的值
func (mdb *MemoryDBDao) Get(key string) (interface{}, bool) {
	mdb.rwLock.RLock()
	defer mdb.rwLock.RUnlock()
	// 先判断过期时间是否存在 如果存在 那键肯定存在 再判断是否过期就行了
	expire, exists := mdb.expires[key]
	if exists {
		if time.Now().After(expire) {
			mdb.deleteKey(key)
			log.Printf("键：%s 在：%v 时已经过期", key, expire)
			return nil, false
		}
		mdb.expires[key] = time.Now().Add(Expiration)
		log.Printf("已延长键：%s 过期时间至：%v", key, mdb.expires[key])
		// 将该键移到 LRU 链表头部
		mdb.lruList.MoveToFront(mdb.lruMap[key])
		return mdb.dataMap[key], true
	}
	//如果过期时间不存在 就判断键是否存在
	value, exists := mdb.dataMap[key]
	if exists {
		// 将该键移到 LRU 链表头部
		mdb.lruList.MoveToFront(mdb.lruMap[key])
	}
	return value, exists
}

// Update 更新键对应的值
func (mdb *MemoryDBDao) Update(key string, value interface{}) bool {
	mdb.rwLock.Lock()
	defer mdb.rwLock.Unlock()
	//先判断过期时间是否存在 如果存在 那键肯定存在 再判断是否过期 不过期就更新
	expire, exists := mdb.expires[key]
	if exists {
		if time.Now().After(expire) {
			mdb.deleteKey(key)
			log.Printf("键：%s 在：%v 时已经过期", key, expire)
			return false
		}
		mdb.expires[key] = time.Now().Add(Expiration)
		log.Printf("已延长键：%s 过期时间至：%v", key, mdb.expires[key])
		mdb.dataMap[key] = value
		log.Printf("修改键：%s 的值为：%v", key, mdb.dataMap[key])
		// 将该键移到 LRU 链表头部
		mdb.lruList.MoveToFront(mdb.lruMap[key])
		return true
	}
	//如果过期时间不存在 就判断键是否存在 再更新
	if _, exists = mdb.dataMap[key]; exists {
		mdb.dataMap[key] = value
		log.Printf("修改键：%s 的值为：%v", key, mdb.dataMap[key])
		// 将该键移到 LRU 链表头部
		mdb.lruList.MoveToFront(mdb.lruMap[key])
		return true
	}
	log.Printf("不存在键：%s", key)
	return false
}

// Delete 删除指定键
func (mdb *MemoryDBDao) Delete(key string) {
	mdb.rwLock.Lock()
	defer mdb.rwLock.Unlock()
	mdb.deleteKey(key)
	log.Printf("删除键: %s", key)
}

// Count 获取数据库中键值对的数量
func (mdb *MemoryDBDao) Count() int {
	mdb.rwLock.RLock()
	defer mdb.rwLock.RUnlock()
	return len(mdb.dataMap)
}

// deleteKey 添加一个不加锁的内部的删除方法  删除过期键和内存淘汰有用
func (mdb *MemoryDBDao) deleteKey(key string) {
	delete(mdb.dataMap, key)
	if _, exists := mdb.expires[key]; exists {
		delete(mdb.expires, key)
	}

	// 从 LRU 链表和映射中移除
	if element, exists := mdb.lruMap[key]; exists {
		mdb.lruList.Remove(element)
		delete(mdb.lruMap, key)
	}
}

// PeriodicDelete 定期删除过期键
func (mdb *MemoryDBDao) PeriodicDelete(examineSize int) {
	mdb.rwLock.Lock()
	defer mdb.rwLock.Unlock()
	//先把所有有过期时间的键取出来
	keys := make([]string, 0, len(mdb.expires))
	for key := range mdb.expires {
		keys = append(keys, key)
	}
	// 随机选择一定数量的键进行检查
	if len(keys) > 0 {
		if len(keys) < examineSize {
			examineSize = len(keys)
		}
		//打乱顺序
		rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
		for _, key := range keys[:examineSize] {
			// 检查键是否过期
			if expire, exists := mdb.expires[key]; exists && time.Now().After(expire) {
				mdb.deleteKey(key)
				log.Printf("定期删除过期键：%s", key)
			}
		}
	}
}

// evict 执行 LRU 淘汰
func (mdb *MemoryDBDao) evict() {
	log.Printf("内存已满(已存储超过：%d个键值对) 通过LRU机制淘汰：%f比例的键", mdb.capacity, mdb.evictRatio)
	// 得到需要淘汰的键的数量
	evictCount := int(float64(mdb.capacity) * mdb.evictRatio)
	if evictCount < 1 {
		log.Printf("淘汰比例过小 已删除最少一个键")
		evictCount = 1
	}
	for i := 0; i < evictCount && mdb.lruList.Len() > 0; i++ {
		// 获取链表尾部元素（最久未使用的键）
		element := mdb.lruList.Back()
		key := element.Value.(string)
		// 删除该键 不能调用带锁的删除方法 不然会死锁 因为Set()方法已经加了写锁 故内部不需要再加锁
		mdb.deleteKey(key)
		log.Printf("LRU 淘汰键：%s", key)
	}
}
