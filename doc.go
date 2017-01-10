/*
Copyright 2017 Turbine Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// The adminserver package provides a tool to wrap a nonstdlib/proc.ManagedProc
// with a simple HTTP server that manages the process lifecycle, including
// termination and signaling it to reload its configuration.
//
// An AdminServer can be constructed directly using adminserver.New(), or
// can be configured using a flag.FlagSet using adminserver.NewFromFlags().
package adminserver
