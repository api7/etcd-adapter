# Copyright api7.ai
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# contianer image registry
REGISTRY ?="api7"

# e2e
E2E_FOCUS ?=""

.PHONY: test
test:
	@go test ./...

.PHONY: bench
bench:
	@go test -bench '^Benchmark' ./...

.PHONY: gofmt
gofmt:
	@find . -name "*.go" | xargs gofmt -w

.PHONY: lint
lint:
	@golangci-lint run

.PHONY: build
build:
	go build \
		-o etcd-adapter \
		main.go

.PHONY: build-image
build-image:
	@docker build -t $(REGISTRY)/etcd-adapter:dev .

.PHONY: e2e-test
e2e-test: build build-image
	cd test/e2e && go mod download && ginkgo -r --focus=$(E2E_FOCUS)