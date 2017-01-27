
[//]: # ( Copyright 2017 Turbine Labs, Inc.                                   )
[//]: # ( you may not use this file except in compliance with the License.    )
[//]: # ( You may obtain a copy of the License at                             )
[//]: # (                                                                     )
[//]: # (     http://www.apache.org/licenses/LICENSE-2.0                      )
[//]: # (                                                                     )
[//]: # ( Unless required by applicable law or agreed to in writing, software )
[//]: # ( distributed under the License is distributed on an "AS IS" BASIS,   )
[//]: # ( WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or     )
[//]: # ( implied. See the License for the specific language governing        )
[//]: # ( permissions and limitations under the License.                      )

# turbinelabs/adminserver

[![GoDoc](https://https://godoc.org/github.com/turbinelabs/adminserver?status.svg)](https://https://godoc.org/github.com/turbinelabs/adminserver)
[https://circleci.com/gh/turbinelabs/adminserver](![CircleCI`](https://circleci.com/gh/turbinelabs/adminserver.svg?style=svg))

The adminserver package provides a tool to wrap a process with a simple HTTP
server that manages the process lifecycle, including termination and signaling.

## Requirements

- Go 1.7.4 or later (previous versions may work, but we don't build or test against them)

## Dependencies

The adminserver project depends on our
[nonstdlib package](https://github.com/turbinelabs/nonstdlib); The tests depend
on our [test package](https://github.com/turbinelabs/test) and
[gomock](https://github.com/golang/mock). It should always be safe to use HEAD
of all master branches of Turbine Labs open source projects together, or to
vendor them with the same git tag.

## Install

```
go get -u github.com/turbinelabs/adminserver/...
```

## Clone/Test

```
mkdir -p $GOPATH/src/turbinelabs
git clone https://github.com/turbinelabs/adminserver.git > $GOPATH/src/turbinelabs/adminserver
go test github.com/turbinelabs/adminserver/...
```

## Godoc

[`adminserver`](https://godoc.org/github.com/turbinelabs/adminserver)

## Versioning

Please see [Versioning of Turbine Labs Open Source Projects](http://github.com/turbinelabs/developer/blob/master/README.md#versioning).

## Pull Requests

Patches accepted! Please see [Contributing to Turbine Labs Open Source Projects](http://github.com/turbinelabs/developer/blob/master/README.md#contributing).
