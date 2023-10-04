/*
Copyright 2019-2023 vChain, Inc.

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

package grpcclient

import "log"

// LcLogger ...
type LcLogger interface {
	Errorf(string, ...interface{})
	Warningf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Close() error
}

type simpleLogger struct{}

func (simpleLogger) Errorf(f string, args ...interface{}) {
	log.Printf("ERROR: "+f, args)
}
func (simpleLogger) Warningf(f string, args ...interface{}) {
	log.Printf("WARN: "+f, args)
}
func (simpleLogger) Infof(f string, args ...interface{}) {
	log.Printf("INFO: "+f, args)
}
func (simpleLogger) Debugf(f string, args ...interface{}) {
	log.Printf("DEBUG: "+f, args)
}
func (simpleLogger) Close() error { return nil }
