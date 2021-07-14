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

import "errors"

var (
	// ErrObjectNotFound means the target is not found from the cache.
	ErrObjectNotFound = errors.New("object not found")
)

// Cache groups all required behaviors that the cache implementations required
// to support the etcd-adapter.
// Currently, the key type is always string.
type Cache interface {
	// GetSingle accepts a key and find the object from the cache.
	// Note if the object is missing, the second return value will
	// be ErrObjectNotFound.
	GetSingle(string) (interface{}, error)
	// GetRange accepts the start key and end key, it will return all objects
	// which key is in the range of this section.
	GetRange(string, string) ([]interface{}, error)
}
