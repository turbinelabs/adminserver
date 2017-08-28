
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

[![Apache 2.0](https://img.shields.io/hexpm/l/plug.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/turbinelabs/adminserver?status.svg)](https://godoc.org/github.com/turbinelabs/adminserver)
[![CircleCI](https://circleci.com/gh/turbinelabs/adminserver.svg?style=shield)](https://circleci.com/gh/turbinelabs/adminserver)
[![Go Report Card](https://goreportcard.com/badge/github.com/turbinelabs/adminserver)](https://goreportcard.com/report/github.com/turbinelabs/adminserver)
[![codecov](https://codecov.io/gh/turbinelabs/adminserver/branch/master/graph/badge.svg)](https://codecov.io/gh/turbinelabs/adminserver)

The adminserver package provides a tool to wrap a process with a simple HTTP
server that manages the process lifecycle, including termination and signaling.

## Requirements

- Go 1.9 or later (previous versions may work, but we don't build or test against them)

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

## Code of Conduct

All Turbine Labs open-sourced projects are released with a
[Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in our
projects you agree to abide by its terms, which will be carefully enforced.
