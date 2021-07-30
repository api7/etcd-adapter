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

package backends

// Item will be used as the key and value type of the backends.
type Item interface {
	// Key returns the unique identical key for this item.
	// Key will be used to decide the partial order. Currently
	// it's string type and cannot be changed.
	Key() string
	// Marshal marshals the item.
	Marshal() ([]byte, error)
}

// Revisioner is the revision manager, revision is a int64 typed integer.
type Revisioner interface {
	// Revision returns the current revision.
	Revision() int64
	// Incr increases the current revision and returns it.
	Incr() int64
}
