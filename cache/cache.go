// Copyright api7.ai
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package cache

// Item will be used as the key and value type of the cache.
type Item interface {
	// Key returns the unique identical key for this item.
	// Key will be used to decide the partial order. Currently
	// it's string type and cannot be changed.
	Key() string
	// Marshal marshals the item.
	Marshal() ([]byte, error)
}

// Cache groups all required behaviors that the cache implementations required
// to support the etcd-adapter. Note this cache interface is special as it doesn't
// have the key or value definitions. The key and value are the object itself as the
// object implements the Item interface so it's self-contained for the partial order.
type Cache interface {
	// Get accepts a key and find the object from the cache.
	// key might not be totally complete but should have enough clues to
	// decide its partial order.
	// It returns nil if then object not found.
	Get(key Item) Item
	// Range accepts the startKey and endKey, it will return all objects
	// which key is in the range of this section (left side inclusive while right side is exclusive).
	// startKey and endKey might not be totally complete but should have enough clues to
	// decide their partial orders.
	// Note both the startKey and endKey should not be nil or the program will
	// panic.
	Range(startKey Item, endKey Item) []Item
	// Put inserts or updates an object.
	Put(object Item)
	// List lists all objects in the cache.
	List() []Item
	// Delete deletes the object from the cache, the given object doesn't have to
	// be totally complete but it should have enough items to keep the partial order.
	Delete(object Item)
}
