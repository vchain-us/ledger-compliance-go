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

// defaultLogger prints only error messages using standard golang logger
type defaultLogger struct{}

func (defaultLogger) Errorf(f string, args ...interface{}) {
	log.Printf("ERROR: "+f, args)
}
func (defaultLogger) Warningf(f string, args ...interface{}) {}
func (defaultLogger) Infof(f string, args ...interface{})    {}
func (defaultLogger) Debugf(f string, args ...interface{})   {}
func (defaultLogger) Close() error                           { return nil }
