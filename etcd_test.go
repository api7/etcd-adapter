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

package etcdadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShowVersion(t *testing.T) {
	w := httptest.NewRecorder()
	a := NewEtcdAdapter(nil).(*adapter)
	a.showVersion(w, nil)

	assert.Equal(t, http.StatusOK, w.Code, "checking status code")
	assert.Equal(t, "{\"etcdserver\":\"3.5.0-pre\",\"etcdcluster\":\"3.5.0\"}", w.Body.String())
}
