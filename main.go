package main

import (
  "sync"
  "fmt"
)


// Type Safe, Thread Safe, nil Safe, reference Safe, wrapper for sync.Map
type SyncMap[K comparable, V any] struct {
  syncMap *sync.Map
  localLock *sync.Mutex
}

// This globalLock and init purely exist so that
// SyncMap can safely be initialized without calling NewSyncMap
// It's easier ergonomically easier but less efficient than NewSyncMap
var globalLock sync.Mutex



func (m *SyncMap[K, V]) init() {
  if m.syncMap == nil {
    globalLock.Lock()
    defer globalLock.Unlock()
    if m.syncMap == nil {
      m.syncMap = &sync.Map{}
      m.localLock = &sync.Mutex{}
    }
  }
}

func (m *SyncMap[K, V]) Clear() {
  m.init()
  m.localLock.Lock()
  defer m.localLock.Unlock()
  m.syncMap.Range(func(key, _ any) bool {
    m.syncMap.Delete(key)
    return true
  })
}

func NewSyncMap[K comparable, V any](maps ...map[K]V) SyncMap[K, V] {
  var sMap SyncMap[K, V]
  sMap.syncMap = &sync.Map{}
  sMap.localLock = &sync.Mutex{}
  for _, m := range maps {
    for key, value := range m {
      sMap.Store(key, value)
    }
  }
  return sMap
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
  m.init()
  value, ok := m.syncMap.Load(key)
  typedValue, typedOk := value.(V)
  return typedValue, (typedOk && ok && value != nil)
}

func (m *SyncMap[K, V]) LoadAndDelete(key K) (V, bool) {
  m.init()
  value, ok := m.syncMap.LoadAndDelete(key)
  typedValue, typedOk := value.(V)
  return typedValue, (typedOk && ok && value != nil)
}

func (m *SyncMap[K, V]) Store(key K, value V) {
  m.init()
  m.syncMap.Store(key, value)
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
  m.init()
  v, ok := m.syncMap.LoadOrStore(key, value)
  typedV, typeOk := v.(V)
  return typedV, ok && typeOk
}

func (m *SyncMap[K, V]) Swap(key K, value V) (previous V, loaded bool) {
  m.init()
  v, ok := m.syncMap.Swap(key, value)
  typedV, typeOk := v.(V)
  return typedV, ok && typeOk
}

func (m *SyncMap[K, V]) Delete(key K) {
  m.init()
  m.syncMap.Delete(key)
}

func (m *SyncMap[K, V]) Range(fn func(key K, value V) bool) {
  m.init()
  m.localLock.Lock()
  defer m.localLock.Unlock()
  wrappedFn := func(key, value any) bool {
    rtrn := true
    (func(){
      defer func(){
        if r:= recover(); r != nil{
          fmt.Println("SyncMap Range Panic: ", r)
        }
      }()
    typedKey, typedKeyOk := key.(K)
    typedValue, typedValueOk := value.(V)
    if typedKeyOk && typedValueOk {
      rtrn = fn(typedKey, typedValue)
    }else {
      fmt.Println("SyncMap: Range assertion failed for key: ", key)
    }
    })()
    return rtrn
  }
  m.syncMap.Range(wrappedFn)
}

func (m *SyncMap[K, V]) ToMap() map[K]V {
  mp := make(map[K]V)
  m.Range(func(key K, value V) bool {
    mp[key] = value
    return true
  })
  return mp
}

func SyncTransform[K1, K2 comparable, V1, V2 any](m1 SyncMap[K1, V1], fn func(key K1, value V1) (K2, V2)) SyncMap[K2, V2] {
  m2 := SyncMap[K2, V2]{}
  m1.Range(func(key K1, value V1) bool {
    k2, v2 := fn(key, value)
    m2.Store(k2, v2)
    return true
  })
  return m2
}

func (m *SyncMap[K, V]) Copy() SyncMap[K, V] {
  return SyncTransform(*m, func(k K, v V) (K, V) { return k, v })
}

func Merge[K comparable, V any](a, b SyncMap[K, V]) SyncMap[K, V] {
  out := SyncMap[K, V]{}
  a.Range(func(k K, v V) bool {
    out.Store(k, v)
    return true
  })
  b.Range(func(k K, v V) bool {
    out.Store(k, v)
    return true
  })
  return out
}

 
