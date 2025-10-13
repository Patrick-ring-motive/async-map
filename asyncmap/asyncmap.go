package asyncmap

import (
	"log"
	"sync"
)

// SyncMap is a Type Safe, Thread Safe, nil Safe, and reference Safe
// generic wrapper around Go's sync.Map.
type SyncMap[K comparable, V any] struct {
	syncMap   *sync.Map
	localLock *sync.Mutex
}

// globalLock is used to safely initialize a zero-value SyncMap instance.
// It is a coarse-grained lock only used once per uninitialized map.
var globalLock sync.Mutex

// lazyInit ensures the underlying syncMap and localLock are initialized.
// It uses a double-checked locking pattern with the package-level globalLock.
func (m *SyncMap[K, V]) lazyInit() {
	if m.syncMap == nil {
		globalLock.Lock()
		defer globalLock.Unlock()
		if m.syncMap == nil {
			m.syncMap = &sync.Map{}
			m.localLock = &sync.Mutex{}
		}
	}
}

// Clear removes all entries from the map.
// It acquires the local lock to ensure atomicity against other composite operations like Range.
func (m *SyncMap[K, V]) Clear() {
	m.lazyInit()
	m.localLock.Lock()
	defer m.localLock.Unlock()
	m.syncMap.Range(func(key, _ any) bool {
		m.syncMap.Delete(key)
		return true
	})
}

// NewSyncMap creates and initializes a new SyncMap, optionally pre-populating it
// with values from the provided maps.
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

// Load returns the value stored in the map for a key, or nil/false if no value is present.
// It enforces type safety and treats stored nil values as "not found".
func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	m.lazyInit()
	value, ok := m.syncMap.Load(key)
	typedValue, typedOk := value.(V)
	// Key must be found (ok), assertion must succeed (typedOk), and value must not be nil
	// (nil check handles stored nil pointers/interfaces).
	return typedValue, (typedOk && ok && value != nil)
}

// Get returns the value for a key, or the zero value of V if the key is not present
// or the stored value is nil/of the wrong type.
func (m *SyncMap[K, V]) Get(key K) V {
	m.lazyInit()
	value, ok := m.syncMap.Load(key)

	var zero V
	if !ok {
		return zero
	}

	if value == nil {
		return zero
	}

	typedValue, typedOk := value.(V)
	if !typedOk {
		return zero
	}

	return typedValue
}

// GetOrDefault returns the value for a key, or the provided defaultValue if the key is not present
// or the stored value is nil/of the wrong type.
// If no defaultValue is provided, the zero value of V is used as the default.
func (m *SyncMap[K, V]) GetOrDefault(key K, defaultValue ...V) V {
	m.lazyInit()
	var df V
	if len(defaultValue) > 0 {
		df = defaultValue[0]
	}
	value, ok := m.syncMap.Load(key)

	// Key not found
	if !ok {
		return df
	}

	// Stored value is nil
	if value == nil {
		return df
	}

	// Type assertion
	typedValue, typedOk := value.(V)
	if !typedOk {
		return df
	}

	return typedValue
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// It enforces type safety and treats stored nil values as "not found".
func (m *SyncMap[K, V]) LoadAndDelete(key K) (V, bool) {
	m.lazyInit()
	value, ok := m.syncMap.LoadAndDelete(key)
	typedValue, typedOk := value.(V)
	return typedValue, (typedOk && ok && value != nil)
}

// Store sets the value for a key.
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.lazyInit()
	m.syncMap.Store(key, value)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	m.lazyInit()
	v, ok := m.syncMap.LoadOrStore(key, value)
	typedV, typeOk := v.(V)
	return typedV, ok && typeOk
}

// Swap stores a new value for a key, and returns the previous value if any.
func (m *SyncMap[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	m.lazyInit()
	v, ok := m.syncMap.Swap(key, value)
	typedV, typeOk := v.(V)
	return typedV, ok && typeOk
}

// Delete deletes the value for a key.
func (m *SyncMap[K, V]) Delete(key K) {
	m.lazyInit()
	m.syncMap.Delete(key)
}

// Range calls fn sequentially for each key and value present in the map.
// If fn returns false, the iteration stops.
// It locks the map locally to prevent concurrent Range/Clear operations.
// It includes a panic recovery block to ensure a panic in the user-supplied fn does not crash the iteration.
func (m *SyncMap[K, V]) Range(fn func(key K, value V) bool) {
	m.lazyInit()
	m.localLock.Lock()
	defer m.localLock.Unlock()

	wrappedFn := func(key, value any) bool {
		rtrn := true
		(func() {
			defer func() {
				if r := recover(); r != nil {
					// NOTE: fmt.Printf has been replaced with log.Printf
					log.Printf("SyncMap Range Panic (Recovered): %+v", r)
				}
			}()
			typedKey, typedKeyOk := key.(K)
			typedValue, typedValueOk := value.(V)
			if typedKeyOk && typedValueOk {
				rtrn = fn(typedKey, typedValue)
			} else {
				// NOTE: fmt.Printf has been replaced with log.Printf (using log.Print for simpler output)
				log.Printf("SyncMap: Range assertion failed for key: %+v", key)
			}
		})()
		return rtrn
	}
	m.syncMap.Range(wrappedFn)
}

// ToMap copies all key/value pairs into a standard Go map.
func (m *SyncMap[K, V]) ToMap() map[K]V {
	mp := make(map[K]V)
	m.Range(func(key K, value V) bool {
		mp[key] = value
		return true
	})
	return mp
}

// SyncTransform creates a new SyncMap by applying a transformation function to all
// elements of the current map.
func SyncTransform[K1, K2 comparable, V1, V2 any](m1 SyncMap[K1, V1], fn func(key K1, value V1) (K2, V2)) SyncMap[K2, V2] {
	m2 := SyncMap[K2, V2]{}
	m1.Range(func(key K1, value V1) bool {
		k2, v2 := fn(key, value)
		m2.Store(k2, v2)
		return true
	})
	return m2
}

// Copy returns a shallow copy of the SyncMap.
func (m *SyncMap[K, V]) Copy() SyncMap[K, V] {
	return SyncTransform(*m, func(k K, v V) (K, V) { return k, v })
}

// Merge combines two SyncMaps into a new SyncMap. Values from b overwrite values from a.
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
